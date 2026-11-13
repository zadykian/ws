package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	// minPaneWidth is the columns under which a pane is too narrow to read beside another, frame
	// included, so that the panes stack.
	minPaneWidth = 24
	// MinPaneHeight and MinFrameWidth are the smallest frame that holds a line of text.
	MinPaneHeight = 3
	// MinFrameWidth is the columns of the smallest frame.
	MinFrameWidth = 3
	// titleHeight is the lines of the title.
	titleHeight = 1
)

// Size is an area of the window, in columns and lines.
type Size struct {
	Width  int
	Height int
}

// Pane is a box of a Layout.
type Pane struct {
	Title string
	Body  string
	// Weight is the pane's part of the room that the panes' least sizes leave. Zero counts as 1.
	Weight int
	// Focused draws the heavier frame, and keeps this pane where the window holds only one.
	Focused bool
}

// Layout is a full screen. The title and its status take the first line, then come the header and
// the panes side by side. What stands below them follows, and the footer's buttons take the last
// lines. A narrow window stacks the panes, and one too short for that shows the focused pane alone.
// A window with no room for a frame shows no pane.
type Layout struct {
	Title  string
	Status string
	// Header is lines across the window, above the panes, such as a screen's tabs. Those it doesn't
	// have take no room.
	Header string
	Panes  []Pane
	// Below is lines across the window, under the panes, such as a screen's strip of detail. Those
	// it doesn't have take no room.
	Below  string
	Footer Footer
}

// PaneSizes returns the room inside each pane's frame, in a window of width and height, so that a
// screen can size what it draws in a pane. A pane the window leaves out has the zero Size.
func (l Layout) PaneSizes(width, height int) []Size {
	sizes, _ := l.frames(width, height)
	for i, size := range sizes {
		if size != (Size{}) {
			sizes[i] = Size{size.Width - 2, size.Height - 2}
		}
	}
	return sizes
}

// View draws the layout as a full screen. The alternate buffer gives the terminal its scrollback
// back at the end. The mouse stays the terminal's, as its capture would block selecting the lines
// a screen prints to copy.
func (l Layout) View(theme Theme, width, height int) tea.View {
	view := tea.NewView(l.Render(theme, width, height))
	view.AltScreen = true
	return view
}

// Render fills a window of width and height, and draws nothing before the window's size is known.
// What doesn't fit is cut at the edges.
func (l Layout) Render(theme Theme, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	parts := []string{l.renderTitle(theme, width)}
	if l.Header != "" {
		parts = append(parts, l.Header)
	}
	if room := l.bodyHeight(width, height); room > 0 {
		panes := l.renderPanes(theme, width, height)
		parts = append(parts, lipgloss.PlaceVertical(room, lipgloss.Top, panes))
	}
	if l.Below != "" {
		parts = append(parts, l.Below)
	}
	if footer := l.Footer.Render(theme, width); footer != "" {
		parts = append(parts, footer)
	}
	return lipgloss.NewStyle().MaxWidth(width).MaxHeight(height).Render(strings.Join(parts, "\n"))
}

// renderTitle draws the title's line, with the status at its right where both fit.
func (l Layout) renderTitle(theme Theme, width int) string {
	text := " " + l.Title
	status := l.Status + " "
	gap := width - lipgloss.Width(text) - lipgloss.Width(status)
	if l.Status == "" || gap < 1 {
		status = ""
		gap = max(width-lipgloss.Width(text), 0)
	}
	return theme.TitleBar.MaxWidth(width).Render(text + strings.Repeat(" ", gap) + status)
}

// renderPanes draws the panes that the window holds, side by side or stacked.
func (l Layout) renderPanes(theme Theme, width, height int) string {
	sizes, stacked := l.frames(width, height)
	boxes := make([]string, 0, len(sizes))
	for i, size := range sizes {
		if size != (Size{}) {
			boxes = append(boxes, l.Panes[i].Render(theme, size))
		}
	}
	if stacked {
		return lipgloss.JoinVertical(lipgloss.Left, boxes...)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, boxes...)
}

// frames returns each pane's size with its frame, and whether the panes stack.
func (l Layout) frames(width, height int) ([]Size, bool) {
	sizes := make([]Size, len(l.Panes))
	room := l.bodyHeight(width, height)
	if len(l.Panes) == 0 || room < MinPaneHeight || width < MinFrameWidth {
		return sizes, true
	}
	if columns := spread(l.Panes, width, minPaneWidth); columns != nil {
		for i, column := range columns {
			sizes[i] = Size{column, room}
		}
		return sizes, false
	}
	if rows := spread(l.Panes, room, MinPaneHeight); rows != nil {
		for i, row := range rows {
			sizes[i] = Size{width, row}
		}
		return sizes, true
	}
	sizes[l.focused()] = Size{width, room}
	return sizes, true
}

// bodyHeight is the lines of the panes, between the header and what stands below them.
func (l Layout) bodyHeight(width, height int) int {
	taken := titleHeight + len(l.Footer.rows(width))
	if l.Header != "" {
		taken += lipgloss.Height(l.Header)
	}
	if l.Below != "" {
		taken += lipgloss.Height(l.Below)
	}
	return max(height-taken, 0)
}

// spread gives each pane least, and the room left by weight, or nil where total can't hold that.
// Each pane's end rounds down, so that the parts sum to total.
func spread(panes []Pane, total, least int) []int {
	left := total - least*len(panes)
	if left < 0 {
		return nil
	}
	sum := 0
	for _, pane := range panes {
		sum += max(pane.Weight, 1)
	}
	parts := make([]int, len(panes))
	weight, given := 0, 0
	for i, pane := range panes {
		weight += max(pane.Weight, 1)
		end := left * weight / sum
		parts[i] = least + end - given
		given = end
	}
	return parts
}

// focused returns the place of the focused pane, or the first where none has the focus.
func (l Layout) focused() int {
	for i, pane := range l.Panes {
		if pane.Focused {
			return i
		}
	}
	return 0
}

// Render draws the pane in a frame of that size, its title in the top edge. A line wider than the
// frame is cut, as a pane holds its shape: a screen wraps by PaneSizes where it wants that.
func (p Pane) Render(theme Theme, frame Size) string {
	edge, border := theme.Frame, theme.Border
	if p.Focused {
		edge, border = theme.FocusFrame, theme.FocusBorder
	}
	title := ""
	if p.Title != "" && frame.Width > 3 {
		title = theme.PaneTitle.MaxWidth(frame.Width - 3).Render(" " + p.Title + " ")
	}
	fill := strings.Repeat(border.Top, max(frame.Width-3-lipgloss.Width(title), 0))
	lines := []string{
		edge.Render(border.TopLeft+border.Top) + title + edge.Render(fill+border.TopRight),
	}
	inner := Size{frame.Width - 2, frame.Height - 2}
	clip := lipgloss.NewStyle().MaxWidth(inner.Width).MaxHeight(inner.Height)
	body := strings.Split(clip.Render(p.Body), "\n")
	for i := range inner.Height {
		line := ""
		if i < len(body) {
			line = body[i]
		}
		pad := strings.Repeat(" ", max(inner.Width-lipgloss.Width(line), 0))
		lines = append(lines, edge.Render(border.Left)+line+pad+edge.Render(border.Right))
	}
	bottom := strings.Repeat(border.Bottom, inner.Width)
	lines = append(lines, edge.Render(border.BottomLeft+bottom+border.BottomRight))
	return strings.Join(lines, "\n")
}
