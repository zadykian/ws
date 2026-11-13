// Package tui runs ws's full-screen terminal UI: the entry points that the commands call, and the
// types that they hand to them. The screens are in internal packages.
package tui

import "github.com/zadykian/ws/engine/tui/internal/widget"

// A Theme styles every part of a screen.
type Theme = widget.Theme

// NewTheme returns the theme for an environment, which getenv reads.
func NewTheme(getenv func(string) string) Theme { return widget.NewTheme(getenv) }

// A FieldSpec describes one field of a sheet.
type FieldSpec = widget.FieldSpec

// The kinds of a sheet's field.
const (
	TextField     = widget.TextField
	PasswordField = widget.PasswordField
	ChoiceField   = widget.ChoiceField
)
