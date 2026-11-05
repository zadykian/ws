package tui

// Pane is a box of a Layout.
type Pane struct {
	Title string
	Body  string
	// Weight is the pane's part of the room that the panes' least sizes leave. Zero counts as 1.
	Weight int
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
