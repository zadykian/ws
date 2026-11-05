package tui

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
)

// twoPanes is a list beside a window of output, which takes the room left twice over.
func twoPanes() Layout {
	return Layout{
		Title:  "ws server apply",
		Status: "3/59",
		Panes: []Pane{
			{Title: "Steps", Body: "a\nb", Focused: true},
			{Title: "Output", Body: "line", Weight: 2},
		},
		Footer: NewFooter(Keys.Quit),
	}
}

func TestSpread(t *testing.T) {
	panes := []Pane{{}, {Weight: 2}}
	cases := []struct {
		total, least int
		want         []int
	}{
		{10, 2, []int{4, 6}},
		{11, 2, []int{4, 7}},
		{4, 2, []int{2, 2}},
		{3, 2, nil},
	}
	for _, c := range cases {
		if got := spread(panes, c.total, c.least); !reflect.DeepEqual(got, c.want) {
			t.Errorf("spread(%d, %d) = %v, want %v", c.total, c.least, got, c.want)
		}
	}
}

func TestPaneSizes(t *testing.T) {
	cases := []struct {
		name          string
		width, height int
		want          []Size
	}{
		{"side by side at 80x24", 80, 24, []Size{{32, 20}, {44, 20}}},
		{"side by side at the narrowest", 48, 24, []Size{{22, 20}, {22, 20}}},
		{"stacked one column under that", 47, 24, []Size{{45, 6}, {45, 12}}},
		{"stacked at the shortest", 40, 8, []Size{{38, 1}, {38, 1}}},
		{"the focused pane alone, too short to stack", 40, 7, []Size{{38, 3}, {}}},
		{"no pane in a window that holds no frame", 80, 4, []Size{{}, {}}},
		{"no pane in a window of no columns", 2, 24, []Size{{}, {}}},
	}
	for _, c := range cases {
		if got := twoPanes().PaneSizes(c.width, c.height); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

func TestPaneSizesKeepTheFocusedPaneInAShortWindow(t *testing.T) {
	layout := twoPanes()
	layout.Panes[0].Focused, layout.Panes[1].Focused = false, true
	want := []Size{{}, {38, 3}}
	if got := layout.PaneSizes(40, 7); !reflect.DeepEqual(got, want) {
		t.Errorf("sizes %v, want %v", got, want)
	}
}

func TestPaneSizesGiveTheFootersRowsTheirRoom(t *testing.T) {
	layout := twoPanes()
	layout.Footer = NewFooter(slices.Repeat([]key.Binding{Keys.Quit}, 6)...)
	want := 24 - 1 - 2 - 2
	if got := layout.PaneSizes(48, 24)[0].Height; got != want {
		t.Errorf("%d lines in a pane, want %d", got, want)
	}
}

func TestPaneRender(t *testing.T) {
	cases := []struct {
		name string
		pane Pane
		want string
	}{
		{"at rest", Pane{Title: "Steps", Body: "a"}, `
╭─ Steps ───╮
│a          │
│           │
╰───────────╯`},
		{"in focus", Pane{Title: "Steps", Body: "a", Focused: true}, `
┏━ Steps ━━━┓
┃a          ┃
┃           ┃
┗━━━━━━━━━━━┛`},
		{"no title", Pane{Body: "a\nb"}, `
╭───────────╮
│a          │
│b          │
╰───────────╯`},
		{"more than it holds", Pane{Title: "A very long title", Body: "0123456789abcdef\nb\nc"}, `
╭─ A very lo╮
│0123456789a│
│b          │
╰───────────╯`},
	}
	for _, c := range cases {
		got := stripped(c.pane.render(NewTheme(noColor("1")), Size{13, 4}))
		if want := strings.TrimPrefix(c.want, "\n"); got != want {
			t.Errorf("%s:\n%s\nwant\n%s", c.name, got, want)
		}
	}
}
