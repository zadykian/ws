package tui

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	return session{screen: recorder{got}, theme: testTheme, window: window}, got
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

// quits says whether cmd ends the program.
func quits(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
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

func TestSessionEndsWithTheWork(t *testing.T) {
	failed := errors.New("a step failed")
	s, _ := testSession()
	s, cmd := feed(s, doneMsg{failed})
	if !quits(cmd) || !s.done || !errors.Is(s.err, failed) {
		t.Errorf("done %v, err %v", s.done, s.err)
	}
}

func TestSessionKeepsAQuestionsKeysFromTheScreen(t *testing.T) {
	s, got := testSession()
	s, _ = feed(s, askMsg{prompt{id: 1, text: "Password", hidden: true}},
		press('a'), tea.PasteMsg{Content: "b"})
	if len(*got) != 0 || s.field.secret != "ab" {
		t.Errorf("the screen got %v, and the field holds %q", *got, s.field.secret)
	}
}

func TestSessionAnswersThroughACmd(t *testing.T) {
	reply := make(chan string, 1)
	s, got := testSession()
	s, cmd := feed(s, askMsg{prompt{id: 1, text: "Host", reply: reply}}, press('a'), enter)
	if len(s.asked) != 0 || cmd == nil {
		t.Fatalf("an answer left %v, and Cmd %v", s.asked, cmd)
	}
	if len(reply) != 0 {
		t.Error("Update replied by itself")
	}
	if msg := cmd(); msg != nil || <-reply != "a" {
		t.Errorf("the reply's Cmd returned %v", msg)
	}
	s, _ = feed(s, press('b'))
	if len(*got) != 1 {
		t.Errorf("with no question the screen got %v", *got)
	}
}

func TestSessionKeepsARefusedQuestion(t *testing.T) {
	s, _ := testSession()
	s, cmd := feed(s, askMsg{prompt{id: 1, text: "Host", valid: refuseEmpty}}, enter)
	if len(s.asked) != 1 || cmd != nil {
		t.Errorf("a refused answer left %v, and Cmd %v", s.asked, cmd)
	}
}

func TestSessionTakesQuestionsInTurn(t *testing.T) {
	s, _ := testSession()
	first, second := prompt{id: 1, text: "First"}, prompt{id: 2, text: "Second"}
	third := prompt{id: 3, text: "Third"}
	s, _ = feed(s, askMsg{first}, askMsg{second}, press('a'), askMsg{third}, retractMsg{2})
	if s.field.prompt.text != "First" || s.field.input.Value() != "a" {
		t.Errorf("a retraction behind the first changed %q", s.field.prompt.text)
	}
	s, _ = feed(s, retractMsg{1})
	if s.field.prompt.text != "Third" || s.field.input.Value() != "" {
		t.Errorf("past the retraction: %q with %q", s.field.prompt.text, s.field.input.Value())
	}
	s, _ = feed(s, retractMsg{9}, retractMsg{3})
	if len(s.asked) != 0 {
		t.Errorf("left %v", s.asked)
	}
}

func TestSessionDrawsOverTheScreen(t *testing.T) {
	s, _ := testSession()
	s, _ = feed(s, askMsg{prompt{id: 1, text: "Host"}})
	view := s.View()
	for _, want := range []string{"Host", "the screen"} {
		if !strings.Contains(view.Content, want) || !view.AltScreen {
			t.Errorf("no %q in:\n%s", want, view.Content)
		}
	}
	if h := lipgloss.Height(view.Content); h != window.Height {
		t.Errorf("drew %d lines over the window's %d", h, window.Height)
	}
}

func TestSessionDrawsInAnyWindow(t *testing.T) {
	s, _ := testSession()
	s, _ = feed(s, askMsg{prompt{id: 1, text: strings.Repeat("a long question ", 20)}})
	for w := range 14 {
		for h := range 10 {
			s, _ = feed(s, tea.WindowSizeMsg{Width: w, Height: h})
			if got := lipgloss.Width(s.View().Content); got > max(w, len("the screen")) {
				t.Errorf("%dx%d: drew %d columns", w, h, got)
			}
		}
	}
}
