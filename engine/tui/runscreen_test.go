package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/zadykian/ws/engine/core/step"
)

var pgUp = tea.KeyPressMsg{Code: tea.KeyPgUp}

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
	screen := NewRunScreen("ws server ubuntu apply", testTheme,
		Group{Name: "base", Steps: []string{"Packages", "Locale"}}, Group{Name: "ssh", Steps: []string{"Keys"}})
	return sent(screen, tea.WindowSizeMsg{Width: 80, Height: 24})
}

func TestRunScreenKnowsEveryStepBeforeTheRun(t *testing.T) {
	got := shown(testRun())
	for _, want := range []string{"ws server ubuntu apply", "0/3", "base", "▸ Packages", "· Locale", "ssh", "· Keys"} {
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
	for _, want := range []string{"1/3", "✓ Packages", "▸ Locale", "base: Locale"} {
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

func TestRunScreenKeysScrollTheWindow(t *testing.T) {
	r := testRun()
	for i := range 100 {
		r = sent(r, Line{Said, "line" + strings.Repeat("x", i%2) + string(rune('0'+i%10))})
	}
	page := r.output.size.Height
	cases := []struct {
		name string
		key  tea.KeyPressMsg
		want int
	}{
		{"up", tea.KeyPressMsg{Code: tea.KeyUp}, 1},
		{"k", tea.KeyPressMsg{Code: 'k', Text: "k"}, 1},
		{"page up", pgUp, page},
		{"ctrl+u", tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}, page},
		{"home", tea.KeyPressMsg{Code: tea.KeyHome}, 100 - page},
		{"g", tea.KeyPressMsg{Code: 'g', Text: "g"}, 100 - page},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := sent(r, c.key).output.back; got != c.want {
				t.Errorf("back = %d, want %d", got, c.want)
			}
		})
	}
	moved := sent(r, tea.KeyPressMsg{Code: tea.KeyHome})
	if got := sent(moved, tea.KeyPressMsg{Code: tea.KeyDown}).output.back; got != 100-page-1 {
		t.Errorf("down: back = %d, want %d", got, 100-page-1)
	}
	if got := sent(moved, tea.KeyPressMsg{Code: tea.KeyPgDown}).output.back; got != 100-2*page {
		t.Errorf("page down: back = %d, want %d", got, 100-2*page)
	}
	if got := sent(moved, tea.KeyPressMsg{Code: tea.KeyEnd}).output.back; got != 0 {
		t.Errorf("end: back = %d, want the window back on the end", got)
	}
}

func TestRunScreenButtonsTakeTheFocus(t *testing.T) {
	r := sent(testRun(), Line{Said, strings.Repeat("line\n", 60)})
	focused := sent(r, tea.KeyPressMsg{Code: tea.KeyTab})
	if focused.footer.Focused() != pageUpPlace {
		t.Fatalf("focus = %d after Tab, want the first button", focused.footer.Focused())
	}
	if got := sent(focused, tea.KeyPressMsg{Code: tea.KeyEnter}).output.back; got != r.output.size.Height {
		t.Errorf("back = %d after Enter on page up, want a page", got)
	}
}

func TestRunScreenFooterShowsTheShortcuts(t *testing.T) {
	got := shown(testRun())
	for _, want := range []string{"[pgup] page up", "[pgdn] page down", "[end] bottom"} {
		if !strings.Contains(got, want) {
			t.Errorf("footer holds no %q:\n%s", want, got)
		}
	}
}

func TestRunScreenTicksFromItsFirstTick(t *testing.T) {
	start := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	r := sent(testRun(), tickMsg{start}, tickMsg{start.Add(75 * time.Second)})
	if got := shown(r); !strings.Contains(got, "01:15") {
		t.Errorf("frame holds no elapsed time:\n%s", got)
	}
	if _, cmd := r.Update(tickMsg{start}); cmd == nil {
		t.Error("a tick asks for no next one")
	}
}

func TestRunScreenInitReadsTheClock(t *testing.T) {
	if _, ok := testRun().Init()().(tickMsg); !ok {
		t.Error("Init's Cmd gives no tick")
	}
}

func TestRunScreenStatusMakesRoomInANarrowWindow(t *testing.T) {
	r := sent(testRun(), tea.WindowSizeMsg{Width: 36, Height: 24})
	if got := r.status(); got != "0/3 00:00" {
		t.Errorf("status = %q, want no bar", got)
	}
	if got := shown(r); !strings.Contains(got, "0/3") {
		t.Errorf("frame holds no count:\n%s", got)
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
