package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/step"
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
	list := queueChoosing(queueOf(testStages()), "Tools")
	q := NewQueueRun("ws mac wizard", testTheme, QueueChoice{left: queueScreen{list: list.filtered("Tools")}})
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

func TestARunOfAQueueFollowsItsStages(t *testing.T) {
	q := queueRunSent(testQueueRun(), step.Event{Kind: step.Done, Step: "Tools"},
		step.Event{Kind: step.Done, Step: "Setup", Index: 1, LeftOut: true},
		step.Event{Kind: step.Started, Step: "Sudo", Index: 2})
	got := queueRunShown(q)
	for _, want := range []string{"2/5", "✓ Tools", "○ Setup", "▸  3 [ ] · Sudo"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in:\n%s", want, got)
		}
	}
}

func TestTheOutputOfAQueueRunFitsItsStrip(t *testing.T) {
	q := testQueueRun()
	if got, want := q.run.output.size, inside(stripSize(Size{100, 30})); got != want {
		t.Errorf("output %v, want %v", got, want)
	}
	if q = queueRunSent(q, haltedMsg{}); q.run.output.size != inside(stripSize(Size{100, 30})) {
		t.Errorf("past a stop, output %v", q.run.output.size)
	}
	if got := queueRunShown(q); !strings.Contains(got, "Stopping after Tools") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestAShortWindowFollowsAQueueRunWithoutItsOutput(t *testing.T) {
	q := queueRunSent(testQueueRun(), tea.WindowSizeMsg{Width: 100, Height: 12}, Line{Said, "installs the tools"})
	if got := queueRunShown(q); strings.Contains(got, "installs the tools") || !strings.Contains(got, "Tools") {
		t.Errorf("frame:\n%s", got)
	}
}

func TestThePagesOfAQueueRunMoveItsStrip(t *testing.T) {
	q := testQueueRun()
	for range 40 {
		q = queueRunSent(q, Line{Said, "installs the tools"})
	}
	if got, want := queueRunSent(q, pgUp).run.output.back, inside(stripSize(Size{100, 30})).Height; got != want {
		t.Errorf("back = %d, want %d", got, want)
	}
}

func TestTheStopButtonOfAQueueRunGoesToTheSession(t *testing.T) {
	q := queueRunSent(testQueueRun(), tab, tab, tab, tab)
	if _, cmd := q.Update(enter); !runs(cmd, stopPressedMsg{}) {
		t.Error("Enter on the stop button asked for no stop")
	}
}
