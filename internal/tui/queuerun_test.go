package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// queueRunSent returns q past msgs.
func queueRunSent(q QueueRun, msgs ...tea.Msg) QueueRun {
	for _, msg := range msgs {
		model, _ := q.Update(msg)
		q = model.(QueueRun)
	}
	return q
}

// queueRunShown is the words of q's frame.
func queueRunShown(q QueueRun) string { return stripped(q.View().Content) }

// testQueueRun follows a queue at 100 by 30, Tools alone on.
func testQueueRun() QueueRun {
	list := choosing(newQueue(testStages()), "Tools")
	q := NewQueueRun("ws mac wizard", testTheme, Choice{list: list.filtered("Tools")})
	return queueRunSent(q, tea.WindowSizeMsg{Width: 100, Height: 30})
}

func TestARunOfAQueueShowsBothListsOverItsOutput(t *testing.T) {
	got := queueRunShown(queueRunSent(testQueueRun(), Line{Said, "installs the tools"}))
	for _, want := range []string{"0/5", " 1 [x] · Tools", " 3 [ ] · Sudo", "installs the tools"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
}
