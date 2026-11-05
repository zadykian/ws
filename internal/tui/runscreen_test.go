package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zadykian/ws/step"
)

// sent gives r each message.
func sent(r RunScreen, msgs ...tea.Msg) RunScreen {
	for _, msg := range msgs {
		model, _ := r.Update(msg)
		r = model.(RunScreen)
	}
	return r
}

// shown is the words of r's frame.
func shown(r RunScreen) string { return stripped(r.View().Content) }

func testRun() RunScreen {
	screen := NewRunScreen("ws server apply", testTheme,
		Group{"base", []string{"Packages", "Locale"}}, Group{"ssh", []string{"Keys"}})
	return sent(screen, tea.WindowSizeMsg{Width: 80, Height: 24})
}

func TestRunScreenKnowsEveryStepBeforeTheRun(t *testing.T) {
	got := shown(testRun())
	for _, want := range []string{"ws server apply", "base", "▸ Packages", "· Locale", "ssh", "· Keys"} {
		if !strings.Contains(got, want) {
			t.Errorf("frame holds no %q:\n%s", want, got)
		}
	}
}

func TestRunScreenFollowsTheEvents(t *testing.T) {
	r := sent(testRun(),
		step.Event{Kind: step.Done, Parent: "base", Step: "Packages"},
		step.Event{Kind: step.Started, Parent: "base", Step: "Locale", Index: 1})
	got := shown(r)
	for _, want := range []string{"✓ Packages", "▸ Locale", "base: Locale"} {
		if !strings.Contains(got, want) {
			t.Errorf("frame holds no %q:\n%s", want, got)
		}
	}
}

func TestRunScreenShowsALineInItsWindow(t *testing.T) {
	if got := shown(sent(testRun(), Line{Piped, "Setting up curl"})); !strings.Contains(got, "Setting up curl") {
		t.Errorf("frame holds no line:\n%s", got)
	}
}

func TestRunScreenFitsItsFrameToTheWindow(t *testing.T) {
	r := sent(testRun(), Line{Said, strings.Repeat("word ", 100)})
	for _, size := range []Size{{80, 24}, {120, 40}, {40, 12}, {30, 8}, {12, 5}} {
		got := sent(r, tea.WindowSizeMsg{Width: size.Width, Height: size.Height}).View().Content
		if lipgloss.Width(got) > size.Width || lipgloss.Height(got) > size.Height {
			t.Errorf("%v: frame is %dx%d", size, lipgloss.Width(got), lipgloss.Height(got))
		}
	}
}

func TestRunScreenWrapsALongLineToTheWindow(t *testing.T) {
	r := sent(testRun(), Line{Said, "first " + strings.Repeat("x", 45) + " last"})
	if got := shown(r); !strings.Contains(got, "last") {
		t.Errorf("frame cut the line:\n%s", got)
	}
}

func TestBar(t *testing.T) {
	cases := []struct {
		done, total, width int
		want               string
	}{
		{0, 4, 4, "░░░░"},
		{1, 4, 4, "█░░░"},
		{4, 4, 4, "████"},
		{0, 0, 4, "░░░░"},
	}
	for _, c := range cases {
		if got := bar(c.done, c.total, c.width); got != c.want {
			t.Errorf("bar(%d, %d, %d) = %q, want %q", c.done, c.total, c.width, got, c.want)
		}
	}
}

func TestElapsed(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "00:00"},
		{59 * time.Second, "00:59"},
		{20*time.Minute + 5*time.Second, "20:05"},
		{time.Hour + 2*time.Minute + 3*time.Second, "1:02:03"},
	}
	for _, c := range cases {
		if got := elapsed(c.d); got != c.want {
			t.Errorf("elapsed(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestRunScreenRunsUnderASession(t *testing.T) {
	s := session{screen: testRun(), theme: testTheme, window: window, cut: testCut}
	s, _ = feed(s, Line{Piped, "a line of output"})
	if got := stripped(s.View().Content); !strings.Contains(got, "a line of output") {
		t.Errorf("frame holds no line:\n%s", got)
	}
}
