// Package tui runs ws's full-screen terminal UI: the entry points that the commands call, and the
// types that they hand to them. The screens are in internal packages.
package tui

import (
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/list"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

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

// A Group is the steps of a role or a stage, by the names that a run's events give them.
type Group = list.Group

// NewGroup names steps as a run of them under parent will.
func NewGroup(parent string, steps []step.Step) Group { return list.NewGroup(parent, steps) }
