package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// wayFile is a file of the form tests' fields.
func wayFile() Sheet { return Sheet{Name: "way.conf", Fields: waySpecs} }

// fed hands s each message in turn, and runs what comes back, as the screen's Cmds leave by its send.
func fed(s configureScreen, msgs ...tea.Msg) configureScreen {
	for _, msg := range msgs {
		next, cmd := s.Update(msg)
		s = next.(configureScreen) //nolint:forcetypeassert // Update returns its own type
		if cmd != nil {
			cmd()
		}
	}
	return s
}

func testConfigure(sheets ...Sheet) configureScreen {
	s := newConfigureScreen("ws client macos configure", testTheme, func(configureJob) {}, nil, sheets)
	return fed(s, tea.WindowSizeMsg{Width: 80, Height: 24})
}

func TestConfigureDrawsAForm(t *testing.T) {
	if got := stripped(testConfigure(wayFile()).View().Content); !strings.Contains(got, "way.conf 1/3") {
		t.Errorf("drew:\n%s", got)
	}
}

func TestConfigureTabWalksOneRingOfFieldsAndButtons(t *testing.T) {
	s := fed(testConfigure(wayFile()), tab, tab, tab)
	if s.footer.Focused() != savePlace || !s.form().Blurred {
		t.Fatalf("past the last field: button %d, blurred = %v", s.footer.Focused(), s.form().Blurred)
	}
	if s = fed(s, tab, tab); s.footer.Focused() >= 0 || s.form().Name() != "HOST" {
		t.Errorf("past the last button: button %d, field %q", s.footer.Focused(), s.form().Name())
	}
	if s = fed(s, shiftTab, shiftTab, shiftTab); s.footer.Focused() >= 0 || s.form().Name() != "USER" {
		t.Errorf("back past the buttons: button %d, field %q", s.footer.Focused(), s.form().Name())
	}
}

func TestConfigureEnterAndTheArrowsStepAlongTheFields(t *testing.T) {
	s := fed(testConfigure(wayFile()), enter, tea.KeyPressMsg{Code: tea.KeyDown})
	if s.form().Name() != "USER" {
		t.Errorf("field %q", s.form().Name())
	}
	if s = fed(s, tea.KeyPressMsg{Code: tea.KeyUp}); s.form().Name() != "PORT" {
		t.Errorf("past Up: field %q", s.form().Name())
	}
}

func TestConfigureLettersOfAKeyTypeInAField(t *testing.T) {
	s := fed(testConfigure(wayFile()), enter, press('j'), press('k'), press('q'))
	if got := s.form().Edits()["PORT"]; got != "jkq" {
		t.Errorf("PORT = %q", got)
	}
}

func TestConfigureAPasteGoesToTheFieldInFocusAlone(t *testing.T) {
	s := fed(testConfigure(wayFile()), enter, tea.PasteMsg{Content: "2222"})
	if got := s.form().Edits()["PORT"]; got != "2222" {
		t.Errorf("PORT = %q", got)
	}
	if s = fed(s, enter, enter, tea.PasteMsg{Content: "x"}); s.form().Edits()["USER"] != "" {
		t.Errorf("a button took the paste: %v", s.form().Edits())
	}
}
