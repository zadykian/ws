package tui

import (
	"maps"
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
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

// cleared is the key that empties a field past its end.
var cleared = tea.KeyPressMsg{Code: 'u', Mod: tea.ModCtrl}

func typedIn(f Form, text string) Form { return f.Update(tea.PasteMsg{Content: text}) }

func moved(t *testing.T, f Form, step int) Form {
	t.Helper()
	f, ok := f.Move(step)
	if !ok {
		t.Fatalf("no field %d from %q", step, f.name())
	}
	return f
}

func TestFormTypesInTheFieldInFocusAlone(t *testing.T) {
	f := typedIn(moved(t, NewForm(testTheme, waySpecs), 1), "2222")
	if got, want := f.Edits(), (map[string]string{"PORT": "2222"}); !maps.Equal(got, want) {
		t.Errorf("Edits = %v, want %v", got, want)
	}
}

func TestFormMoveStopsAtEitherEnd(t *testing.T) {
	f := NewForm(testTheme, waySpecs)
	if _, ok := f.Move(-1); ok {
		t.Error("the focus moved above the first field")
	}
	f = moved(t, moved(t, f, 1), 1)
	if _, ok := f.Move(1); ok || f.name() != "USER" {
		t.Errorf("Move = %v, focus on %q", ok, f.name())
	}
}

func TestFormHidesAFieldAndBringsItBackWithItsTyping(t *testing.T) {
	f := moved(t, typedIn(moved(t, NewForm(testTheme, waySpecs), 1), "2222"), -1)
	f = typedIn(f.Update(cleared), "none")
	if want := []string{"HOST", "USER"}; !slices.Equal(f.names, want) {
		t.Errorf("names = %v, want %v", f.names, want)
	}
	f = f.Update(cleared)
	if got := f.fields["PORT"].Value(); got != "2222" || !slices.Contains(f.names, "PORT") {
		t.Errorf("names = %v, PORT = %q", f.names, got)
	}
}

func TestFormKeepsTheFocusOnItsFieldAcrossAFlow(t *testing.T) {
	f := typedIn(NewForm(testTheme, waySpecs).Update(cleared), "none")
	if f.name() != "HOST" {
		t.Errorf("focus on %q", f.name())
	}
}

func TestFormOlderValueKeepsItsField(t *testing.T) {
	before := NewForm(testTheme, waySpecs)
	typedIn(before, "x")
	if len(before.Edits()) != 0 {
		t.Errorf("Edits = %v", before.Edits())
	}
}

func TestFormBlurLeavesNoFieldTheKeys(t *testing.T) {
	f := typedIn(NewForm(testTheme, waySpecs).Blur(), "x")
	if len(f.Edits()) != 0 {
		t.Errorf("Edits = %v", f.Edits())
	}
	if f = typedIn(f.Enter(true), "ada"); f.Edits()["USER"] != "ada" {
		t.Errorf("past Enter(true): Edits = %v", f.Edits())
	}
	if f = f.Blur().Enter(false); f.name() != "HOST" {
		t.Errorf("past Enter(false): focus on %q", f.name())
	}
}
