package tui

import "charm.land/lipgloss/v2"

// boxWidth is the most columns of a box drawn over the screen.
const boxWidth = 64

// box draws body in a focused frame that fits it, or nothing where the window holds no frame.
func box(theme Theme, title, body string, window Size) string {
	if window.Width < minFrameWidth || window.Height < minPaneHeight {
		return ""
	}
	frame := Size{min(window.Width, boxWidth), min(lipgloss.Height(body)+2, window.Height)}
	return Pane{Title: title, Body: body, Focused: true}.render(theme, frame)
}
