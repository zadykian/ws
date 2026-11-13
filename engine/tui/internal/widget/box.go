package widget

import "charm.land/lipgloss/v2"

// BoxWidth is the most columns of a box drawn over the screen.
const BoxWidth = 64

// Box draws body in a focused frame that fits it, or nothing where the window holds no frame.
func Box(theme Theme, title, body string, window Size) string {
	if window.Width < MinFrameWidth || window.Height < MinPaneHeight {
		return ""
	}
	frame := Size{min(window.Width, BoxWidth), min(lipgloss.Height(body)+2, window.Height)}
	return Pane{Title: title, Body: body, Focused: true}.Render(theme, frame)
}
