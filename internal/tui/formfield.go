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
	// Label is a question's words, or a file's key. Help says more, under the value.
	Label string
	Help  string
	// Value is what a text or a choice starts with. A password starts empty, as a form shows no secret.
	Value string
	// Default is what an empty text gives, and the option that a choice with no Value starts on.
	Default string
	// Options are a choice's values, in the order they show. A choice starts on its first where its
	// Value, or its Default with no Value, names none.
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
	// chosen is the place of a choice's option, and start the value before any edit.
	chosen int
	start  string
	// problem is why Valid refuses the value, once an edit or a Check found that: an empty field that
	// nobody touched shows none.
	problem error
}

// NewFormField returns the field of spec, which takes no key before Focus.
func NewFormField(spec FieldSpec) FormField {
	input := newInput()
	input.Blur()
	if spec.Kind == TextField {
		input.SetValue(spec.Value)
	}
	chosen := max(slices.Index(spec.Options, cmp.Or(spec.Value, spec.Default)), 0)
	f := FormField{spec: spec, input: input, chosen: chosen}
	f.start = f.Value()
	return f
}

// Respec gives the field what spec asks now, as a value above it changed that, and keeps what was
// typed. A field nobody edited follows the new default, and a refusal on show is checked again. A
// choice keeps its options.
func (f FormField) Respec(spec FieldSpec) FormField {
	edited := f.Edited()
	f.spec = spec
	if !edited {
		f.start = f.Value()
	}
	if f.problem != nil {
		f, _ = f.Check()
	}
	return f
}

// Focus gives the field the keys.
func (f FormField) Focus() FormField {
	_ = f.input.Focus()
	return f
}

// Blur takes the keys from the field.
func (f FormField) Blur() FormField {
	f.input.Blur()
	return f
}

// Fit sets the columns of the field, past which it scrolls. The input scrolls only as its cursor
// moves, and from the end alone does it work out a narrower view, so the cursor goes there and back.
func (f FormField) Fit(width int) FormField {
	f.input.SetWidth(max(width-lipgloss.Width(f.input.Prompt)-1, 1))
	at := f.input.Position()
	f.input.CursorEnd()
	f.input.SetCursor(at)
	return f
}

// Update takes a key or a paste, and checks a value that changed. The input's Cmd goes: it starts a
// blink, or reads a clipboard that a server lacks, into a message that would reach the screen. A
// terminal's own paste comes as tea.PasteMsg.
func (f FormField) Update(msg tea.Msg) FormField {
	if !f.input.Focused() {
		return f
	}
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

// Edited reports whether the value differs from the one the field started on, so that a form writes
// what the maintainer changed alone, and a choice's first option never stands in for a file's value.
func (f FormField) Edited() bool {
	return f.Value() != f.start
}

// Check runs Valid on the value, and keeps its refusal for the field to show.
func (f FormField) Check() (FormField, bool) {
	f.problem = nil
	if f.spec.Valid != nil {
		f.problem = f.spec.Valid(f.Value())
	}
	return f, f.problem == nil
}

// view draws the line that takes the keys. A choice in focus shows its keys as arrows. Too narrow for
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
	before, after := "  ", "  "
	if f.input.Focused() {
		before, after = "← ", " →"
	}
	if line := before + strings.Join(cells, " ") + after; lipgloss.Width(line) <= width {
		return line
	}
	return before + chosen + after
}

// label is the field's words, with the default that an empty text gives.
func (f FormField) label() string {
	if f.spec.Kind == TextField && f.spec.Default != "" {
		return f.spec.Label + " [" + f.spec.Default + "]"
	}
	return f.spec.Label
}

// Lines draws the label, the line that takes the keys, the help and why the value is refused, in at
// most room lines. The line that takes the keys stays whatever the room, and the refusal goes after
// the words around it, as it says what to mend.
func (f FormField) Lines(theme Theme, width, room int) []string {
	if room < 1 {
		return nil
	}
	room--
	take := func(style lipgloss.Style, words string) []string {
		if words == "" {
			return nil
		}
		lines := strings.Split(style.Width(max(width, 1)).Render(words), "\n")
		lines = lines[:min(room, len(lines))]
		room -= len(lines)
		return lines
	}
	var why []string
	if f.problem != nil {
		why = take(theme.Bad, f.problem.Error())
	}
	label := take(lipgloss.NewStyle(), f.label())
	help := take(theme.Muted, f.spec.Help)
	return slices.Concat(label, []string{f.view(theme, width)}, help, why)
}
