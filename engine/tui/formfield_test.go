package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func choice(fallback string) FormField {
	spec := FieldSpec{Kind: ChoiceField, Options: []string{"lan", "cloud", "mac-vm"}, Default: fallback}
	return NewFormField(spec).Focus()
}

func TestChoiceMovesRoundItsOptions(t *testing.T) {
	cases := []struct {
		name string
		keys []tea.Msg
		want string
	}{
		{"right", []tea.Msg{right}, "cloud"},
		{"space", []tea.Msg{space}, "cloud"},
		{"past the last", []tea.Msg{right, right, right}, "lan"},
		{"before the first", []tea.Msg{left}, "mac-vm"},
		{"h and l", []tea.Msg{press('l'), press('l'), press('h')}, "cloud"},
		{"another key", []tea.Msg{press('x'), tea.PasteMsg{Content: "cloud"}}, "lan"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := choice("")
			for _, msg := range c.keys {
				f = f.Update(msg)
			}
			if got := f.Value(); got != c.want {
				t.Errorf("value = %q, want %q", got, c.want)
			}
		})
	}
}

func TestChoiceStartsOnItsDefault(t *testing.T) {
	for fallback, want := range map[string]string{"": "lan", "cloud": "cloud", "gone": "lan"} {
		if got := choice(fallback).Value(); got != want {
			t.Errorf("default %q: the value is %q, want %q", fallback, got, want)
		}
	}
}

func TestChoiceChecksTheOptionItMovesTo(t *testing.T) {
	spec := FieldSpec{Kind: ChoiceField, Options: []string{"a", "b"}, Valid: func(v string) error {
		if v == "b" {
			return errors.New("not b")
		}
		return nil
	}}
	f := NewFormField(spec).Focus()
	if f = f.Update(right); f.problem == nil {
		t.Error("the refused option shows no problem")
	}
	if f = f.Update(right); f.problem != nil {
		t.Errorf("past the refused option: %v", f.problem)
	}
}

func TestChoiceWithNoOptionsHoldsNothing(t *testing.T) {
	f := NewFormField(FieldSpec{Kind: ChoiceField}).Focus().Update(right)
	if got := f.Value(); got != "" {
		t.Errorf("value = %q", got)
	}
	if got := f.view(testTheme, 20); strings.Contains(got, "[") {
		t.Errorf("drew an option: %q", got)
	}
}

func TestChoiceShowsItsOptionsOrTheChosenOneAlone(t *testing.T) {
	f := choice("cloud")
	if got := f.view(testTheme, 40); !strings.Contains(got, "[cloud]") || !strings.Contains(got, "mac-vm") {
		t.Errorf("wide: %q", got)
	}
	got := f.view(testTheme, 12)
	if !strings.Contains(got, "[cloud]") || strings.Contains(got, "lan") || lipgloss.Width(got) > 12 {
		t.Errorf("narrow: %q", got)
	}
}

func TestFieldStartsOnItsValue(t *testing.T) {
	cases := []struct {
		name string
		spec FieldSpec
		want string
	}{
		{"a text", FieldSpec{Value: "ws", Default: "other"}, "ws"},
		{"a choice", FieldSpec{Kind: ChoiceField, Options: []string{"a", "b"}, Value: "b", Default: "a"}, "b"},
		{"a password", FieldSpec{Kind: PasswordField, Value: "s3cret"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := NewFormField(c.spec).Value(); got != c.want {
				t.Errorf("value = %q, want %q", got, c.want)
			}
		})
	}
}

func TestFieldLinesGiveUpTheirWordsBeforeTheInput(t *testing.T) {
	refused := func(string) error { return errors.New("refused") }
	f := NewFormField(FieldSpec{Label: "KEY", Help: "says more", Value: "held", Valid: refused}).Fit(30)
	f, _ = f.Check()
	cases := []struct {
		room int
		want []string
	}{
		{0, nil},
		{1, []string{"held"}},
		{2, []string{"held", "refused"}},
		{3, []string{"KEY", "held", "refused"}},
		{9, []string{"KEY", "held", "says more", "refused"}},
	}
	for _, c := range cases {
		got := f.Lines(testTheme, 30, c.room)
		if len(got) != len(c.want) {
			t.Fatalf("room %d: %d lines: %q", c.room, len(got), got)
		}
		for i, want := range c.want {
			if !strings.Contains(got[i], want) {
				t.Errorf("room %d, line %d: no %q in %q", c.room, i, want, got[i])
			}
		}
	}
}

func TestFieldNamesADefaultOnATextAlone(t *testing.T) {
	text := NewFormField(FieldSpec{Label: "Host", Default: "ws"})
	if got := text.Lines(testTheme, 30, 3)[0]; !strings.Contains(got, "Host [ws]") {
		t.Errorf("a text's label: %q", got)
	}
	picked := NewFormField(FieldSpec{Kind: ChoiceField, Label: "Target", Options: []string{"a"}, Default: "a"})
	if got := picked.Lines(testTheme, 30, 3)[0]; strings.Contains(got, "[a]") {
		t.Errorf("a choice's label: %q", got)
	}
}

func TestBlurredFieldTakesNoKey(t *testing.T) {
	specs := map[string]FieldSpec{
		"a text":     {},
		"a password": {Kind: PasswordField},
		"a choice":   {Kind: ChoiceField, Options: []string{"a", "b"}},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			f := NewFormField(spec).Focus().Blur()
			want := f.Value()
			if got := f.Update(press('x')).Update(right).Update(tea.PasteMsg{Content: "y"}).Value(); got != want {
				t.Errorf("value = %q, want %q", got, want)
			}
		})
	}
}

func TestBlurredFieldShowsNoColorAndNoArrows(t *testing.T) {
	theme := NewTheme(func(string) string { return "yes" })
	text := NewFormField(FieldSpec{Value: "held"}).Fit(20)
	if got := text.view(theme, 20); !strings.Contains(got, "held") || strings.Contains(got, "\x1b[3") {
		t.Errorf("a text at rest: %q", got)
	}
	if got := choice("").Blur().view(theme, 40); strings.ContainsAny(got, "←→") {
		t.Errorf("a choice at rest: %q", got)
	}
}

func TestFieldEditedComparesTheValueWithItsStart(t *testing.T) {
	text := NewFormField(FieldSpec{Value: "ws"}).Focus()
	if text.Edited() {
		t.Error("a text edited before the first key")
	}
	if text = text.Update(press('x')); !text.Edited() {
		t.Error("a typed value counts as no edit")
	}
	if text = text.Update(tea.KeyPressMsg{Code: tea.KeyBackspace}); text.Edited() {
		t.Error("a value back at its start counts as an edit")
	}
	unknown := NewFormField(FieldSpec{Kind: ChoiceField, Options: []string{"a", "b"}, Value: "gone"}).Focus()
	if unknown.Edited() {
		t.Error("a file's unknown value counts as an edit")
	}
	if unknown = unknown.Update(right); !unknown.Edited() {
		t.Error("a moved choice counts as no edit")
	}
}

func TestRespecKeepsWhatWasTypedAndFollowsADefault(t *testing.T) {
	typed := NewFormField(FieldSpec{Default: "22"}).Focus().Update(tea.PasteMsg{Content: "2222"})
	if got := typed.Respec(FieldSpec{Default: "22022"}); got.Value() != "2222" || !got.Edited() {
		t.Errorf("typed: Value = %q, Edited = %v", got.Value(), got.Edited())
	}
	left := NewFormField(FieldSpec{Default: "22"}).Respec(FieldSpec{Default: "22022"})
	if left.Value() != "22022" || left.Edited() {
		t.Errorf("left alone: Value = %q, Edited = %v", left.Value(), left.Edited())
	}
}

func TestRespecChecksARefusalAgain(t *testing.T) {
	refused := errors.New("outside the LAN")
	f, ok := NewFormField(FieldSpec{Valid: func(string) error { return refused }}).Check()
	if ok {
		t.Fatal("Check took the value")
	}
	f = f.Respec(FieldSpec{})
	if got := strings.Join(f.Lines(testTheme, 40, 5), "\n"); strings.Contains(got, refused.Error()) {
		t.Errorf("a mended value still shows its refusal:\n%s", got)
	}
}
