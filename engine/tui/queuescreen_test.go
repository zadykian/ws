package tui

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/run"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// testQueueScreen is the two lists at 100 by 30, their scan under way.
func testQueueScreen() queueScreen {
	s := openQueue("ws client macos wizard", testTheme, testStages(), make(chan QueueChoice, 1))
	return queueSent(s, tea.WindowSizeMsg{Width: 100, Height: 30})
}

func TestTheTwoListsStandSideBySideOverTheDetail(t *testing.T) {
	got := queueShown(testQueueScreen())
	for _, want := range []string{"ws client macos wizard", "scanning 0/5", "Mac", "ws: roles not scanned", "Detail",
		"▸  1 [ ] · Tools", "   3 [ ] · Sudo", "Not scanned yet."} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	lines := strings.Split(got, "\n")
	if i := strings.Index(lines[2], "Tools"); i < 0 || !strings.Contains(lines[2][i:], "Sudo") {
		t.Errorf("not side by side:\n%s", got)
	}
}

func TestAShortWindowHasNoStrip(t *testing.T) {
	got := queueShown(queueSent(testQueueScreen(), tea.WindowSizeMsg{Width: 100, Height: 12}))
	if strings.Contains(got, "Detail") || !strings.Contains(got, "Sudo") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestANarrowWindowStacksTheLists(t *testing.T) {
	got := queueShown(queueSent(testQueueScreen(), tea.WindowSizeMsg{Width: 40, Height: 30}))
	if strings.Index(got, "Tools") > strings.Index(got, "Sudo") || !strings.Contains(got, "Detail") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestTheFilterShowsOverTheListInFocus(t *testing.T) {
	s := queueSent(testQueueScreen(), press('/'), press('t'), press('w'), press('o'))
	got := queueShown(s)
	if !strings.Contains(got, "/ two") || !strings.Contains(got, "No step matches.") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestAStripTakesAThirdOfTheWindowWithinBounds(t *testing.T) {
	cases := []struct {
		height, want int
	}{{13, 0}, {14, stripLeast}, {24, 8}, {60, stripMost}}
	for _, c := range cases {
		if got := stripSize(widget.Size{Width: 80, Height: c.height}).Height; got != c.want {
			t.Errorf("a window of %d lines has a strip of %d, want %d", c.height, got, c.want)
		}
	}
}

func TestAReadRoleClearsTheTitleOfWsList(t *testing.T) {
	stages := []Stage{{Name: "Run", Side: "ws", Nested: []list.Group{{Name: "base"}}}}
	l := queueOf(stages)
	if got := l.panes()[l.pane("ws")].Title; got != "ws: roles not scanned" {
		t.Errorf("title = %q", got)
	}
	if got := l.reported("Run", nil).panes()[l.pane("ws")].Title; got != "ws" {
		t.Errorf("title = %q", got)
	}
}

// wsStages is a queue of ws server ubuntu wizard, whose every stage runs on ws.
func wsStages() []Stage {
	return []Stage{
		{Name: "Values", Side: "ws"},
		{Name: "Run", Side: "ws", Waits: "Values", Nested: []list.Group{{Name: "base"}}},
		{Name: "Signing", Side: "ws", Waits: "Run"},
	}
}

func TestStagesAllOnWsStandInOneList(t *testing.T) {
	s := openQueue("ws server ubuntu wizard", testTheme, wsStages(), make(chan QueueChoice, 1))
	s = queueSent(s, tea.WindowSizeMsg{Width: 100, Height: 30}, list.ScannedMsg{}, press('/'), press('s'))
	got := queueShown(s)
	for _, want := range []string{"ws server ubuntu wizard", "ws", "/ s", "   3 [x] · Signing", "Detail"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Mac") || s.list.side != "ws" || len(s.layout().Panes) != 1 {
		t.Errorf("side %q, frame:\n%s", s.list.side, got)
	}
}

func TestTabLeavesOneListForTheButtons(t *testing.T) {
	s := openQueue("ws server ubuntu wizard", testTheme, wsStages(), make(chan QueueChoice, 1))
	s = queueSent(s, tea.WindowSizeMsg{Width: 100, Height: 30}, list.ScannedMsg{}, tab)
	if s.list.side != "ws" || s.footer.Focused() != list.PickTogglePlace {
		t.Errorf("side %q, focus %d", s.list.side, s.footer.Focused())
	}
	if s = queueSent(s, shiftTab); s.list.side != "ws" || s.footer.Focused() >= 0 {
		t.Errorf("side %q, focus %d", s.list.side, s.footer.Focused())
	}
	if s = queueSent(s, shiftTab); s.footer.Focused() != queueConfigurePlace || !strings.Contains(queueShown(s), "configure") {
		t.Errorf("focus %d, frame:\n%s", s.footer.Focused(), queueShown(s))
	}
}

// chosenBy returns what s gives past msgs, with the choice's channel read.
func chosenBy(t *testing.T, msgs ...tea.Msg) QueueChoice {
	picks := make(chan QueueChoice, 1)
	s := queueSent(openQueue("ws", testTheme, testStages(), picks), tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, msg := range msgs {
		model, cmd := s.Update(msg)
		s = model.(queueScreen)
		if cmd != nil {
			cmd()
			return <-picks
		}
	}
	t.Fatalf("no choice past %v", msgs)
	return QueueChoice{}
}

func TestTabWalksTheTwoListsThenTheButtons(t *testing.T) {
	s := queueSent(testQueueScreen(), list.ScannedMsg{}, tab)
	if s.list.side != "ws" || s.footer.Focused() >= 0 {
		t.Errorf("side %q, focus %d", s.list.side, s.footer.Focused())
	}
	if s = queueSent(s, tab); s.list.side != "ws" || s.footer.Focused() != list.PickTogglePlace {
		t.Errorf("side %q, focus %d", s.list.side, s.footer.Focused())
	}
	s = queueSent(s, shiftTab, shiftTab)
	if s.list.side != "Mac" || s.footer.Focused() >= 0 {
		t.Errorf("side %q, focus %d", s.list.side, s.footer.Focused())
	}
	if s = queueSent(s, shiftTab); s.footer.Focused() != queueConfigurePlace {
		t.Errorf("focus %d", s.footer.Focused())
	}
	if s = queueSent(s, tab); s.list.side != "Mac" || s.footer.Focused() >= 0 {
		t.Errorf("side %q, focus %d", s.list.side, s.footer.Focused())
	}
}

func TestAScansEndOpensTheQueueChosen(t *testing.T) {
	got := chosenBy(t, list.ScannedMsg{}, press('c'))
	if got.Next != NextCheck || len(got.Only) != 5 || got.Partial != nil {
		t.Errorf("choice = %+v", got)
	}
}

func TestConfigureGoesBeforeTheScanEnds(t *testing.T) {
	if got := chosenBy(t, press('e')); got.Next != NextConfigure || got.Only != nil {
		t.Errorf("choice = %+v", got)
	}
}

func TestPendingTurnsAWholeStageThatIsDue(t *testing.T) {
	s := queueSent(testQueueScreen(), planned("", "Setup", 1, oneChange("a")), list.ScannedMsg{}, press('n'), press('p'))
	if got := s.list.only(); !slices.Equal(got, []string{"Tools", "Setup"}) {
		t.Errorf("only = %q", got)
	}
	if !s.list.stages[0].row.On() || !s.list.stages[1].row.On() {
		t.Errorf("stages = %+v", s.list.stages[:2])
	}
}

func TestAReportFillsWsList(t *testing.T) {
	sections := []StateSection{{Name: "base", Reports: []step.Report{{Step: "Packages"}}}}
	got := queueShown(queueSent(testQueueScreen(), run.ReportedMsg{Stage: "Run", Sections: sections}))
	if strings.Contains(got, "not scanned") || !strings.Contains(got, "✓ Packages") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestARefusedReportLeavesTheRolesAndSaysWhy(t *testing.T) {
	got := queueShown(queueSent(testQueueScreen(), run.ReportedMsg{Stage: "Run", Problem: "ws on ws is another version"}))
	if !strings.Contains(got, "ws: roles not scanned") || !strings.Contains(got, "Not scanned: ws on ws is another version") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestAStageInPartSaysThatItStaysPending(t *testing.T) {
	s := queueSent(testQueueScreen(), list.ScannedMsg{})
	s.list, s.at = queueChoosing(s.list, "one"), withCursor(s.at, "Mac", 1)
	if got := queueShown(s); !strings.Contains(got, "Runs in part, so it stays pending.") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestPickStagesEndsTheScanWithTheChoice(t *testing.T) {
	c := noKeys(io.Discard)
	c.In = strings.NewReader("e")
	got, err := PickStages(t.Context(), c, "ws", testStages(), waitForCtx)
	if err != nil || got.Next != NextConfigure {
		t.Errorf("PickStages = %+v, %v", got, err)
	}
}

func TestStateKeyNamesTheListInFocus(t *testing.T) {
	if got := chosenBy(t, list.ScannedMsg{}, press('s')); got.Next != NextState || got.Side != "Mac" || len(got.Only) != 0 {
		t.Errorf("s on the Mac's list = %+v", got)
	}
	if got := chosenBy(t, list.ScannedMsg{}, tab, press('s')); got.Next != NextState || got.Side != "ws" {
		t.Errorf("s on ws's list = %+v", got)
	}
}

func TestStateKeyWaitsForTheScanAndTheFilter(t *testing.T) {
	if s := queueSent(testQueueScreen(), press('s')); s.Left {
		t.Error("s left a scan under way")
	}
	s := queueSent(testQueueScreen(), list.ScannedMsg{}, press('/'), press('s'))
	if s.Left || s.list.words != "s" {
		t.Errorf("left %v, words %q", s.Left, s.list.words)
	}
}

// A list of steps alone has no machine to read, and no such button.
func TestStateKeyIsAQueuesAlone(t *testing.T) {
	s := pickSent(newPickScreen("ws", testTheme, nil, make(chan Choice, 1)), list.ScannedMsg{}, press('s'))
	if s.Left {
		t.Error("s left a list of steps")
	}
}

// What the maintainer chose comes back: a new scan would choose the boxes again.
func TestReopenKeepsTheBoxesAndRunsNoScan(t *testing.T) {
	s := queueSent(testQueueScreen(), list.ScannedMsg{Problem: "no network"}, press('n'), press(' '))
	var out bytes.Buffer
	c := noKeys(&out)
	c.In = strings.NewReader("\r")
	got, err := Reopen(t.Context(), c, QueueChoice{Next: NextState, left: s})
	if err != nil || !slices.Equal(got.Only, []string{"Tools"}) {
		t.Errorf("Reopen = %+v, %v", got, err)
	}
}

func TestEachSideKeepsItsCursor(t *testing.T) {
	s := queueSent(testQueueScreen(), list.ScannedMsg{}, press('j'), press('j'), press('j'), tab)
	if s.list.side != "ws" || s.at["Mac"] != 3 || s.at["ws"] != 0 {
		t.Errorf("side %q, cursors %v", s.list.side, s.at)
	}
	if got := strings.Count(queueShown(s), "▸"); got != 1 {
		t.Errorf("%d cursors drawn, want the focused list's alone", got)
	}
	if s = queueSent(s, shiftTab); s.list.side != "Mac" || s.at["Mac"] != 3 {
		t.Errorf("side %q, cursors %v", s.list.side, s.at)
	}
}

func TestAMoveLeavesTheScreenItCameFromItsCursors(t *testing.T) {
	was := queueSent(testQueueScreen(), list.ScannedMsg{}, press('j'))
	moved := queueSent(was, press('j'), tab)
	if was.at["Mac"] != 1 || was.at["ws"] != 0 || moved.at["Mac"] != 2 {
		t.Errorf("cursors %v before, %v after", was.at, moved.at)
	}
}

func TestThreeSidesDrawThreePanesAndTabWalksThem(t *testing.T) {
	third := "VM"
	stages := []Stage{{Name: "One", Side: "ws"}, {Name: "Two", Side: "Mac"}, {Name: "Three", Side: third}, {Name: "Four", Side: "ws"}}
	s := queueSent(openQueue("ws", testTheme, stages, make(chan QueueChoice, 1)), tea.WindowSizeMsg{Width: 120, Height: 30})
	if got := s.list.sides(); !slices.Equal(got, []string{"ws", "Mac", third}) || len(s.list.panes()) != 3 {
		t.Fatalf("sides %v, %d panes: want ws, Mac and the third, in the order named", got, len(s.list.panes()))
	}
	var seen []string
	for range 3 {
		seen = append(seen, s.list.side)
		s = queueSent(s, tab)
	}
	if !slices.Equal(seen, []string{"ws", "Mac", third}) || s.footer.Focused() < 0 {
		t.Errorf("Tab met %v, then button %d", seen, s.footer.Focused())
	}
}

func TestAQueueWithNoStageHasNoSide(t *testing.T) {
	if q := queueOf(nil); len(q.sides()) != 0 || len(q.panes()) != 0 {
		t.Errorf("sides %v, panes %v", q.sides(), q.panes())
	}
}
