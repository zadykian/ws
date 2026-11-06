package tui

import (
	"maps"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// Form is the fields of a file with one in focus, a value that a screen's Update replaces. Which
// fields show follows the edits, as an answer decides the next question.
type Form struct {
	theme Theme
	// specs gives the fields that show, under the edits so far. It runs on each edit.
	specs func(edits map[string]string) []FieldSpec
	// names is the labels of the fields that show, in order. fields holds each field that has shown,
	// so that one that hides and shows again keeps what was typed in it.
	names  []string
	fields map[string]FormField
	// focus is the place in names of the field in focus, and blurred says that a button has the keys.
	focus   int
	blurred bool
	// top is the place of the first field that draws, and size the room that the fields draw in.
	top  int
	size Size
}

// NewForm returns the form of specs, with the focus on its first field.
func NewForm(theme Theme, specs func(edits map[string]string) []FieldSpec) Form {
	return Form{theme: theme, specs: specs, fields: map[string]FormField{}}.flow()
}

// name is the label of the field in focus, or nothing in a form with none.
func (f Form) name() string {
	if len(f.names) == 0 {
		return ""
	}
	return f.names[f.focus]
}

// flow asks which fields show under the edits, and keeps the focus on its field.
func (f Form) flow() Form {
	current := f.name()
	specs := f.specs(f.Edits())
	f.fields = maps.Clone(f.fields)
	f.names = make([]string, 0, len(specs))
	for _, spec := range specs {
		field, known := f.fields[spec.Label]
		if known {
			field = field.Respec(spec)
		} else {
			field = NewFormField(spec).Fit(f.size.Width)
		}
		f.fields[spec.Label] = field
		f.names = append(f.names, spec.Label)
	}
	if place := slices.Index(f.names, current); place >= 0 {
		return f.settle(place)
	}
	return f.settle(f.focus)
}

// settle gives the field at place the keys, and it alone, unless a button has them.
func (f Form) settle(place int) Form {
	f.focus = max(min(place, len(f.names)-1), 0)
	f.fields = maps.Clone(f.fields)
	for name, field := range f.fields {
		f.fields[name] = field.Blur()
	}
	if name := f.name(); name != "" && !f.blurred {
		f.fields[name] = f.fields[name].Focus()
	}
	return f.inSight()
}

// inSight scrolls no further than the field in focus needs to draw whole.
func (f Form) inSight() Form {
	f.top = min(f.top, f.focus)
	for f.top < f.focus && f.height(f.top, f.focus) > f.size.Height {
		f.top++
	}
	return f
}

// height is the lines from the field at one place through the field at another.
func (f Form) height(from, through int) int {
	lines := -1
	for _, name := range f.names[from : through+1] {
		lines += len(f.block(name)) + 1
	}
	return lines
}

// block draws a field.
func (f Form) block(name string) []string {
	return f.fields[name].Lines(f.theme, f.size.Width, f.size.Height)
}

// Blur gives the keys up to a button.
func (f Form) Blur() Form {
	f.blurred = true
	return f.settle(f.focus)
}

// Enter takes the keys back from a button, on the last field or the first.
func (f Form) Enter(last bool) Form {
	f.blurred = false
	if last {
		return f.settle(len(f.names) - 1)
	}
	return f.settle(0)
}

// Fit sets the room that the fields draw in.
func (f Form) Fit(size Size) Form {
	f.size, f.top = size, 0
	f.fields = maps.Clone(f.fields)
	for name, field := range f.fields {
		f.fields[name] = field.Fit(size.Width)
	}
	return f.inSight()
}

// Render draws the fields from the top one on, with a line between two, as far as the room goes.
func (f Form) Render() string {
	var lines []string
	for _, name := range f.names[f.top:] {
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, f.block(name)...)
	}
	return strings.Join(lines[:min(len(lines), f.size.Height)], "\n")
}

// Move steps the focus along the fields, and reports false past either end, which is a button's.
func (f Form) Move(step int) (Form, bool) {
	place := f.focus + step
	if place < 0 || place >= len(f.names) {
		return f, false
	}
	return f.settle(place), true
}

// Update hands the field in focus a key or a paste, and flows again where its value changed.
func (f Form) Update(msg tea.Msg) Form {
	name := f.name()
	if name == "" {
		return f
	}
	before := f.fields[name]
	f.fields = maps.Clone(f.fields)
	f.fields[name] = before.Update(msg)
	if f.fields[name].Value() != before.Value() {
		return f.flow()
	}
	return f
}

// Edits returns the value of each field that differs from what it started on, by label. A field that
// no longer shows counts, so that specs reads it as it comes back, and a save skips what doesn't show.
func (f Form) Edits() map[string]string {
	edits := map[string]string{}
	for name, field := range f.fields {
		if field.Edited() {
			edits[name] = field.Value()
		}
	}
	return edits
}
