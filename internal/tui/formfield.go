package tui

import (
	"cmp"
	"slices"
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
	// ChoiceField takes one of its spec's Options.
	ChoiceField
)

// The keys of a choice. Up and Down stay a form's, to move between its fields.
var (
	optionBack = key.NewBinding(key.WithKeys("left", "h"))
	optionNext = key.NewBinding(key.WithKeys("right", "l", "space"))
)

// FieldSpec is what a field asks for.
type FieldSpec struct {
	Kind FieldKind
	// Default is what an empty text gives, and the option that a choice starts on.
	Default string
	// Options are a choice's values, in the order they show. A choice starts on its first where
	// Default names none.
	Options []string
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
	// chosen is the place of a choice's option.
	chosen int
	// problem is why Valid refuses the value, once an edit or a Check found that: an empty field that
	// nobody touched shows none.
	problem error
}

// NewFormField returns the field of spec, which takes no key before Focus.
func NewFormField(spec FieldSpec) FormField {
	input := newInput()
	input.Blur()
	chosen := max(slices.Index(spec.Options, spec.Default), 0)
	return FormField{spec: spec, input: input, chosen: chosen}
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
	switch f.spec.Kind {
	case PasswordField:
		f = f.edit(msg)
	case ChoiceField:
		f = f.choose(msg)
	default:
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

// choose moves a choice along its options, round the ends.
func (f FormField) choose(msg tea.Msg) FormField {
	press, ok := msg.(tea.KeyPressMsg)
	count := len(f.spec.Options)
	switch {
	case !ok || count == 0:
	case key.Matches(press, optionBack):
		f.chosen = (f.chosen + count - 1) % count
	case key.Matches(press, optionNext):
		f.chosen = (f.chosen + 1) % count
	}
	return f
}

// Value returns what the field holds, as effect.Terminal reads it: a password as typed, a choice's
// option, and a text trimmed, an empty one taking the default.
func (f FormField) Value() string {
	switch {
	case f.spec.Kind == PasswordField:
		return f.secret
	case f.spec.Kind != ChoiceField:
		return cmp.Or(strings.TrimSpace(f.input.Value()), f.spec.Default)
	case len(f.spec.Options) == 0:
		return ""
	}
	return f.spec.Options[f.chosen]
}

// Check runs Valid on the value, and keeps its refusal for the field to show.
func (f FormField) Check() (FormField, bool) {
	f.problem = nil
	if f.spec.Valid != nil {
		f.problem = f.spec.Valid(f.Value())
	}
	return f, f.problem == nil
}

// view draws the line that takes the keys. A choice's arrows show the keys it takes. Too narrow for
// its options, it shows the chosen one alone. Its brackets mark that one under NO_COLOR.
func (f FormField) view(theme Theme, width int) string {
	if f.spec.Kind != ChoiceField {
		return f.input.View()
	}
	cells := make([]string, 0, len(f.spec.Options))
	chosen := ""
	for i, option := range f.spec.Options {
		cell := " " + option + " "
		if i == f.chosen {
			chosen = theme.Shortcut.Render("[" + option + "]")
			cell = chosen
		}
		cells = append(cells, cell)
	}
	if line := "← " + strings.Join(cells, " ") + " →"; lipgloss.Width(line) <= width {
		return line
	}
	return "← " + chosen + " →"
}
