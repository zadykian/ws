package tui

import (
	"slices"
	"testing"
)

// waySpecs asks for a port where the host has a value, as a wizard's question hangs on the one above
// it.
func waySpecs(edits map[string]string) []FieldSpec {
	if edits["HOST"] == "none" {
		return []FieldSpec{{Label: "HOST", Value: "ws.example"}, {Label: "USER"}}
	}
	return []FieldSpec{{Label: "HOST", Value: "ws.example"}, {Label: "PORT", Default: "22"}, {Label: "USER"}}
}

func TestNewFormShowsItsFieldsWithTheFocusOnTheFirst(t *testing.T) {
	f := NewForm(testTheme, waySpecs)
	if want := []string{"HOST", "PORT", "USER"}; !slices.Equal(f.names, want) {
		t.Errorf("names = %v, want %v", f.names, want)
	}
	if f.name() != "HOST" || len(f.Edits()) != 0 {
		t.Errorf("focus on %q, Edits = %v", f.name(), f.Edits())
	}
}

func TestFormWithNoFieldHasNoFocus(t *testing.T) {
	f := NewForm(testTheme, func(map[string]string) []FieldSpec { return nil })
	if f.name() != "" {
		t.Errorf("focus on %q", f.name())
	}
}
