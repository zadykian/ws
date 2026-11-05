package tui

import (
	"reflect"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// recorder is a screen that keeps what it gets.
type recorder struct{ got *[]tea.Msg }

func (r recorder) Init() tea.Cmd { return nil }

func (r recorder) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	*r.got = append(*r.got, msg)
	return r, nil
}

func (r recorder) View() tea.View { return tea.NewView("the screen") }

func testSession() (session, *[]tea.Msg) {
	got := &[]tea.Msg{}
	return session{screen: recorder{got}, window: window}, got
}

// feed gives s each message, and returns the last Cmd.
func feed(s session, msgs ...tea.Msg) (session, tea.Cmd) {
	var cmd tea.Cmd
	for _, msg := range msgs {
		var model tea.Model
		model, cmd = s.Update(msg)
		s = model.(session)
	}
	return s, cmd
}

func TestSessionHandsTheScreenWhatIsNotARuns(t *testing.T) {
	s, got := testSession()
	size := tea.WindowSizeMsg{Width: 100, Height: 30}
	msgs := []tea.Msg{Line{Said, "a line"}, press('a'), tea.PasteMsg{Content: "b"}, size}
	s, _ = feed(s, msgs...)
	if !reflect.DeepEqual(*got, msgs) || s.window != (Size{100, 30}) {
		t.Errorf("the screen got %v, and the window is %v", *got, s.window)
	}
}

func TestSessionHoldsToTheAlternateBuffer(t *testing.T) {
	s, _ := testSession()
	if view := s.View(); !view.AltScreen || view.Content != "the screen" {
		t.Errorf("view %+v", view)
	}
}
