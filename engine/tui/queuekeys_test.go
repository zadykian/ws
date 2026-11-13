package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// queueFrame is the queue's screen at 100 by 30, its scan under way, with no footer.
func queueFrame() queueScreen {
	return queueScreen{
		listScreen: listScreen{title: "ws client macos wizard", theme: testTheme, window: Size{100, 30}, scanning: true},
		list:       queueOf(testStages()),
		footer:     NewFooter(append(listButtons(), stateButton, configureButton)...),
		picks:      make(chan QueueChoice, 1),
	}
}

func TestTheQueueScreenDrawsItsListsOverTheDetail(t *testing.T) {
	got := stripped(queueFrame().View().Content)
	for _, want := range []string{"ws client macos wizard", "scanning 0/5", "Mac", "ws: roles not scanned", "Detail",
		"▸  1 [ ] · Tools", "   3 [ ] · Sudo", "Not scanned yet."} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
}

func TestTheQueueScreenGivesItsChoiceAndWaitsForTheWorkToEnd(t *testing.T) {
	picks := make(chan QueueChoice, 1)
	s := queueFrame()
	s.picks = picks
	s, leaving := s.leave(s.queueChoice(NextCheck))
	leaving()
	if got := <-picks; got.Next != NextCheck || !s.left {
		t.Errorf("choice = %+v, left %v", got, s.left)
	}
}

func TestTheQueueScreenLeavesForTheStateOfTheListInFocus(t *testing.T) {
	picks := make(chan QueueChoice, 1)
	s := queueFrame()
	s.picks = picks
	s, _ = s.pressed(tab)
	s, leaving := s.pressed(press('s'))
	leaving()
	if got := <-picks; got.Next != NextState || got.Side != "ws" || !s.left {
		t.Errorf("choice = %+v, left %v", got, s.left)
	}
}

// queueShown is the words of s's frame.
func queueShown(s queueScreen) string { return stripped(s.View().Content) }

// queueSent returns s past msgs.
func queueSent(s queueScreen, msgs ...tea.Msg) queueScreen {
	for _, msg := range msgs {
		model, _ := s.Update(msg)
		s = model.(queueScreen)
	}
	return s
}

func TestTheQueueScreenHoldsItsBoxesUntilTheScanEnds(t *testing.T) {
	s := openQueue("ws", testTheme, testStages(), make(chan QueueChoice, 1))
	if s = queueSent(s, press('a')); len(s.list.only()) != 0 {
		t.Errorf("only = %q during the scan", s.list.only())
	}
	s = queueSent(s, scannedMsg{}, press('a'))
	if s.scanning || len(s.list.only()) != 5 {
		t.Errorf("scanning %v, only = %q", s.scanning, s.list.only())
	}
}
