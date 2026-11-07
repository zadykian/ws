package tui

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/step"
)

// testQueueScreen is the two lists at 100 by 30, their scan under way.
func testQueueScreen() pickScreen {
	s := newQueueScreen("ws mac wizard", testTheme, testStages(), make(chan Choice, 1))
	return pickSent(s, tea.WindowSizeMsg{Width: 100, Height: 30})
}

func TestTheTwoListsStandSideBySideOverTheDetail(t *testing.T) {
	got := pickShown(testQueueScreen())
	for _, want := range []string{"ws mac wizard", "scanning 0/5", "Mac", "ws: roles not scanned", "Detail",
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
	got := pickShown(pickSent(testQueueScreen(), tea.WindowSizeMsg{Width: 100, Height: 12}))
	if strings.Contains(got, "Detail") || !strings.Contains(got, "Sudo") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestANarrowWindowStacksTheLists(t *testing.T) {
	got := pickShown(pickSent(testQueueScreen(), tea.WindowSizeMsg{Width: 40, Height: 30}))
	if strings.Index(got, "Tools") > strings.Index(got, "Sudo") || !strings.Contains(got, "Detail") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestTheFilterShowsOverTheListInFocus(t *testing.T) {
	s := pickSent(testQueueScreen(), press('/'), press('t'), press('w'), press('o'))
	got := pickShown(s)
	if !strings.Contains(got, "/ two") || !strings.Contains(got, "No step matches.") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestAStripTakesAThirdOfTheWindowWithinBounds(t *testing.T) {
	cases := []struct {
		height, want int
	}{{13, 0}, {14, stripLeast}, {24, 8}, {60, stripMost}}
	for _, c := range cases {
		if got := stripSize(Size{80, c.height}).Height; got != c.want {
			t.Errorf("a window of %d lines has a strip of %d, want %d", c.height, got, c.want)
		}
	}
}

func TestAReadRoleClearsTheTitleOfWsList(t *testing.T) {
	stages := []Stage{{Name: "Run", Side: WsSide, Nested: []Group{{Name: "base"}}}}
	l := newQueue(stages)
	if got := l.panes()[l.pane(WsSide)].Title; got != "ws: roles not scanned" {
		t.Errorf("title = %q", got)
	}
	if got := l.reported("Run", nil).panes()[l.pane(WsSide)].Title; got != "ws" {
		t.Errorf("title = %q", got)
	}
}

// wsStages is a queue of ws server wizard, whose every stage runs on ws.
func wsStages() []Stage {
	return []Stage{
		{Name: "Values", Side: WsSide},
		{Name: "Run", Side: WsSide, Waits: "Values", Nested: []Group{{Name: "base"}}},
		{Name: "Signing", Side: WsSide, Waits: "Run"},
	}
}

func TestStagesAllOnWsStandInOneList(t *testing.T) {
	s := newQueueScreen("ws server wizard", testTheme, wsStages(), make(chan Choice, 1))
	s = pickSent(s, tea.WindowSizeMsg{Width: 100, Height: 30}, scannedMsg{}, press('/'), press('s'))
	got := pickShown(s)
	for _, want := range []string{"ws server wizard", "ws", "/ s", "   3 [x] · Signing", "Detail"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "Mac") || s.list.side != WsSide || len(s.layout().Panes) != 1 {
		t.Errorf("side %d, frame:\n%s", s.list.side, got)
	}
}

func TestTabLeavesOneListForTheButtons(t *testing.T) {
	s := newQueueScreen("ws server wizard", testTheme, wsStages(), make(chan Choice, 1))
	s = pickSent(s, tea.WindowSizeMsg{Width: 100, Height: 30}, scannedMsg{}, tab)
	if s.list.side != WsSide || s.footer.Focused() != pickTogglePlace {
		t.Errorf("side %d, focus %d", s.list.side, s.footer.Focused())
	}
	if s = pickSent(s, shiftTab); s.list.side != WsSide || s.footer.Focused() >= 0 {
		t.Errorf("side %d, focus %d", s.list.side, s.footer.Focused())
	}
}

// chosenBy returns what s gives past msgs, with the choice's channel read.
func chosenBy(t *testing.T, msgs ...tea.Msg) Choice {
	picks := make(chan Choice, 1)
	s := pickSent(newQueueScreen("ws", testTheme, testStages(), picks), tea.WindowSizeMsg{Width: 100, Height: 30})
	for _, msg := range msgs {
		model, cmd := s.Update(msg)
		s = model.(pickScreen)
		if cmd != nil {
			cmd()
			return <-picks
		}
	}
	t.Fatalf("no choice past %v", msgs)
	return Choice{}
}

func TestTabWalksTheTwoListsThenTheButtons(t *testing.T) {
	s := pickSent(testQueueScreen(), scannedMsg{}, tab)
	if s.list.side != WsSide || s.footer.Focused() >= 0 {
		t.Errorf("side %d, focus %d", s.list.side, s.footer.Focused())
	}
	if s = pickSent(s, tab); s.list.side != WsSide || s.footer.Focused() != pickTogglePlace {
		t.Errorf("side %d, focus %d", s.list.side, s.footer.Focused())
	}
	s = pickSent(s, shiftTab, shiftTab)
	if s.list.side != MacSide || s.footer.Focused() >= 0 {
		t.Errorf("side %d, focus %d", s.list.side, s.footer.Focused())
	}
	if s = pickSent(s, shiftTab); s.footer.Focused() != pickStatePlace {
		t.Errorf("focus %d", s.footer.Focused())
	}
	if s = pickSent(s, tab); s.list.side != MacSide || s.footer.Focused() >= 0 {
		t.Errorf("side %d, focus %d", s.list.side, s.footer.Focused())
	}
}

func TestAScansEndOpensTheQueueChosen(t *testing.T) {
	got := chosenBy(t, scannedMsg{}, press('c'))
	if !got.Check || len(got.Only) != 5 || got.Partial != nil {
		t.Errorf("choice = %+v", got)
	}
}

func TestConfigureGoesBeforeTheScanEnds(t *testing.T) {
	if got := chosenBy(t, press('e')); !got.Configure || got.Only != nil {
		t.Errorf("choice = %+v", got)
	}
}

func TestPendingTurnsAWholeStageThatIsDue(t *testing.T) {
	s := pickSent(testQueueScreen(), planned("", "Setup", 1, oneChange("a")), scannedMsg{}, press('n'), press('p'))
	if got := s.list.only(); !slices.Equal(got, []string{"Tools", "Setup"}) {
		t.Errorf("only = %q", got)
	}
	if !s.list.rows[2].on() || !s.list.rows[3].on() {
		t.Errorf("rows = %+v", s.list.rows[2:4])
	}
}

func TestAReportFillsWsList(t *testing.T) {
	sections := []StateSection{{Name: "base", Reports: []step.Report{{Step: "Packages"}}}}
	got := pickShown(pickSent(testQueueScreen(), reportedMsg{stage: "Run", sections: sections}))
	if strings.Contains(got, "not scanned") || !strings.Contains(got, "✓ Packages") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestARefusedReportLeavesTheRolesAndSaysWhy(t *testing.T) {
	got := pickShown(pickSent(testQueueScreen(), reportedMsg{stage: "Run", problem: "ws on ws is another version"}))
	if !strings.Contains(got, "ws: roles not scanned") || !strings.Contains(got, "Not scanned: ws on ws is another version") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestAStageInPartSaysThatItStaysPending(t *testing.T) {
	s := pickSent(testQueueScreen(), scannedMsg{})
	s.list = choosing(s.list, "one")
	s.list = atRow(s.list, "Setup")
	if got := pickShown(s); !strings.Contains(got, "Runs in part, so it stays pending.") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestPickStagesEndsTheScanWithTheChoice(t *testing.T) {
	c := noKeys(io.Discard)
	c.In = strings.NewReader("e")
	got, err := PickStages(t.Context(), c, "ws", testStages(), waitForCtx)
	if err != nil || !got.Configure {
		t.Errorf("PickStages = %+v, %v", got, err)
	}
}

func TestStateKeyNamesTheListInFocus(t *testing.T) {
	if got := chosenBy(t, scannedMsg{}, press('s')); !got.State || got.Side != MacSide || len(got.Only) != 0 {
		t.Errorf("s on the Mac's list = %+v", got)
	}
	if got := chosenBy(t, scannedMsg{}, tab, press('s')); !got.State || got.Side != WsSide {
		t.Errorf("s on ws's list = %+v", got)
	}
}

func TestStateKeyWaitsForTheScanAndTheFilter(t *testing.T) {
	if s := pickSent(testQueueScreen(), press('s')); s.left {
		t.Error("s left a scan under way")
	}
	s := pickSent(testQueueScreen(), scannedMsg{}, press('/'), press('s'))
	if s.left || s.list.words != "s" {
		t.Errorf("left %v, words %q", s.left, s.list.words)
	}
}

// A list of steps alone has no machine to read, and no such button.
func TestStateKeyIsAQueuesAlone(t *testing.T) {
	s := pickSent(newPickScreen("ws", testTheme, nil, make(chan Choice, 1)), scannedMsg{}, press('s'))
	if s.left {
		t.Error("s left a list of steps")
	}
}

// What the maintainer chose comes back: a new scan would choose the boxes again.
func TestReopenKeepsTheBoxesAndRunsNoScan(t *testing.T) {
	s := pickSent(testQueueScreen(), scannedMsg{problem: "no network"}, press('n'), press(' '))
	var out bytes.Buffer
	c := noKeys(&out)
	c.In = strings.NewReader("\r")
	got, err := Reopen(t.Context(), c, Choice{State: true, back: s})
	if err != nil || !slices.Equal(got.Only, []string{"Tools"}) {
		t.Errorf("Reopen = %+v, %v", got, err)
	}
}

func TestABridgeReportReachesTheScreen(t *testing.T) {
	b, sent := testBridge(io.Discard)
	b.Report("Run", nil, "refused")
	if got := (*sent)[0].(reportedMsg); got.stage != "Run" || got.problem != "refused" {
		t.Errorf("sent %+v", got)
	}
}
