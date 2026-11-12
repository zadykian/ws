package tui

import (
	"strings"
	"testing"
)

// queueFrame is the queue's screen at 100 by 30, its scan under way, with no footer.
func queueFrame() queueScreen {
	return queueScreen{
		listScreen: listScreen{title: "ws mac wizard", theme: testTheme, window: Size{100, 30}, scanning: true},
		list:       newQueue(testStages()),
	}
}

func TestTheQueueScreenDrawsItsListsOverTheDetail(t *testing.T) {
	got := stripped(queueFrame().View().Content)
	for _, want := range []string{"ws mac wizard", "scanning 0/5", "Mac", "ws: roles not scanned", "Detail",
		"▸  1 [ ] · Tools", "   3 [ ] · Sudo", "Not scanned yet."} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
}
