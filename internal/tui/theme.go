// Package tui holds the parts of ws's full-screen terminal UI: the theme, the layout and the keys.
// A screen built on them is a pure Update and View, so that tests compare values and text.
package tui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme is the look that the screens share, each style named by what it marks. It uses the 16
// ANSI colors alone: the terminal's palette sets them, so they suit its background, and
// Terminal.app, which has 256 colors, shows them.
type Theme struct {
	// Muted is text that steps back. It is faint rather than bright black, which some palettes
	// draw in the background's color.
	Muted lipgloss.Style
	// Good, Warn and Bad mark what went well, what needs a look, and what failed.
	Good lipgloss.Style
	Warn lipgloss.Style
	Bad  lipgloss.Style
}

// NewTheme builds the theme for the environment that getenv reads. NO_COLOR counts here, with any
// value as no-color.org says, as Bubble Tea takes it for a boolean and would miss NO_COLOR=yes.
func NewTheme(getenv func(string) string) Theme {
	plain := lipgloss.NewStyle()
	paint := func(style lipgloss.Style, ansi color.Color) lipgloss.Style {
		if getenv("NO_COLOR") != "" {
			return style
		}
		return style.Foreground(ansi)
	}
	return Theme{
		Muted: plain.Faint(true),
		Good:  paint(plain, lipgloss.Green),
		Warn:  paint(plain, lipgloss.Yellow),
		Bad:   paint(plain.Bold(true), lipgloss.Red),
	}
}
