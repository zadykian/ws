package tui

const (
	// minPaneWidth is the columns under which a pane is too narrow to read beside another, frame
	// included, so that the panes stack.
	minPaneWidth = 24
	// minPaneHeight and minFrameWidth are the smallest frame that holds a line of text.
	minPaneHeight = 3
	minFrameWidth = 3
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

// Layout is a full screen: the title and its status on the first line, the panes side by side,
// and the footer's buttons on the last lines. A narrow window stacks the panes, and one too short
// for that shows the focused pane alone. A window with no room for a frame shows no pane.
type Layout struct {
	Title  string
	Status string
	Panes  []Pane
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

// frames returns each pane's size with its frame, and whether the panes stack.
func (l Layout) frames(width, height int) ([]Size, bool) {
	sizes := make([]Size, len(l.Panes))
	room := l.bodyHeight(width, height)
	if len(l.Panes) == 0 || room < minPaneHeight || width < minFrameWidth {
		return sizes, true
	}
	if columns := spread(l.Panes, width, minPaneWidth); columns != nil {
		for i, column := range columns {
			sizes[i] = Size{column, room}
		}
		return sizes, false
	}
	if rows := spread(l.Panes, room, minPaneHeight); rows != nil {
		for i, row := range rows {
			sizes[i] = Size{width, row}
		}
		return sizes, true
	}
	sizes[l.focused()] = Size{width, room}
	return sizes, true
}

// bodyHeight is the lines between the title and the footer.
func (l Layout) bodyHeight(width, height int) int {
	return max(height-titleHeight-len(l.Footer.rows(width)), 0)
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
