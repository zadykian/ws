package tui

import (
	"cmp"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// FieldKind says how a field takes its value.
type FieldKind int

// The kinds of field.
const (
	// TextField takes a line, and shows it.
	TextField FieldKind = iota
	// PasswordField takes a line, and shows nothing of it.
	PasswordField
)

// FieldSpec is what a field asks for.
type FieldSpec struct {
	Kind FieldKind
	// Default is what an empty text gives.
	Default string
	// Valid is pure, as it runs on each edit. A nil Valid takes any value.
	Valid func(string) error
}

// FormField is a field of a form: a value that Update replaces, as a screen's model is.
type FormField struct {
	spec  FieldSpec
	input textinput.Model
	// secret is a password. The input never holds it: its sanitizer makes a space of a tab or a line
	// end, which a value that shows nothing would carry unseen.
	secret string
	// problem is why Valid refuses the value, once an edit or a Check found that: an empty field that
	// nobody touched shows none.
	problem error
}

// NewFormField returns the field of spec, which takes no key before Focus.
func NewFormField(spec FieldSpec) FormField {
	input := newInput()
	input.Blur()
	return FormField{spec: spec, input: input}
}

// Focus gives the field the keys.
func (f FormField) Focus() FormField {
	_ = f.input.Focus()
	return f
}

// Fit sets the columns of the field, past which it scrolls.
func (f FormField) Fit(width int) FormField {
	f.input.SetWidth(max(width-lipgloss.Width(f.input.Prompt)-1, 1))
	return f
}

// Update takes a key or a paste, and checks a value that changed. The input's Cmd goes: it starts a
// blink, or reads a clipboard that a server lacks, into a message that would reach the screen. A
// terminal's own paste comes as tea.PasteMsg.
func (f FormField) Update(msg tea.Msg) FormField {
	before := f.Value()
	if f.spec.Kind == PasswordField {
		f = f.edit(msg)
	} else {
		f.input, _ = f.input.Update(msg)
	}
	if f.Value() != before {
		f, _ = f.Check()
	}
	return f
}

// edit takes a key or a paste into a password, as a terminal's line takes them for the plain prompt:
// Backspace and Ctrl-U erase, and a paste gives its first line. Enter alone answers, as nothing shows
// what a paste brought.
func (f FormField) edit(msg tea.Msg) FormField {
	switch msg := msg.(type) {
	case tea.PasteMsg:
		line, _, _ := strings.Cut(strings.ReplaceAll(msg.Content, "\r", "\n"), "\n")
		f.secret += line
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, f.input.KeyMap.DeleteBeforeCursor):
			f.secret = ""
		case key.Matches(msg, f.input.KeyMap.DeleteCharacterBackward):
			_, size := utf8.DecodeLastRuneInString(f.secret)
			f.secret = f.secret[:len(f.secret)-size]
		default:
			f.secret += msg.Text
		}
	}
	return f
}

// Value returns what the field holds, as effect.Terminal reads it: a password as typed, and a text
// trimmed, an empty one taking the default.
func (f FormField) Value() string {
	if f.spec.Kind == PasswordField {
		return f.secret
	}
	return cmp.Or(strings.TrimSpace(f.input.Value()), f.spec.Default)
}

// Check runs Valid on the value, and keeps its refusal for the field to show.
func (f FormField) Check() (FormField, bool) {
	f.problem = nil
	if f.spec.Valid != nil {
		f.problem = f.spec.Valid(f.Value())
	}
	return f, f.problem == nil
}
