package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// wayFile is a file of the form tests' fields.
func wayFile() Sheet { return Sheet{Name: "way.conf", Fields: waySpecs} }

func testConfigure(sheets ...Sheet) configureScreen {
	s, _ := newConfigureScreen("ws mac configure", testTheme, sheets).Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	return s.(configureScreen) //nolint:forcetypeassert // Update returns its own type
}

func TestConfigureDrawsAForm(t *testing.T) {
	if got := stripped(testConfigure(wayFile()).View().Content); !strings.Contains(got, "way.conf 1/3") {
		t.Errorf("drew:\n%s", got)
	}
}
