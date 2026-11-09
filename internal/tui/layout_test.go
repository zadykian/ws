package tui

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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

func TestRenderAt80x24(t *testing.T) {
	lines := strings.Split(stripped(twoPanes().Render(NewTheme(noColor("1")), 80, 24)), "\n")
	if want := " ws server apply" + strings.Repeat(" ", 59) + "3/59 "; lines[0] != want {
		t.Errorf("title %q, want %q", lines[0], want)
	}
	if !strings.HasPrefix(lines[1], "┏━ Steps ") || !strings.Contains(lines[1], "╭─ Output ") {
		t.Errorf("panes not side by side: %q", lines[1])
	}
	if got := strings.TrimRight(lines[23], " "); got != "[q] quit" {
		t.Errorf("last line %q", got)
	}
}

func TestRenderStacksInANarrowWindow(t *testing.T) {
	got := stripped(twoPanes().Render(NewTheme(noColor("1")), 30, 24))
	first := strings.Index(got, "┏━ Steps ")
	second := strings.Index(got, "╭─ Output ")
	if first < 0 || second < first || strings.Count(got[:second], "\n") < 2 {
		t.Errorf("panes not stacked:\n%s", got)
	}
}

func TestRenderDrawsTheHeaderAboveThePanes(t *testing.T) {
	layout := twoPanes()
	layout.Header = "tab one\ntab two"
	lines := strings.Split(layout.Render(testTheme, 80, 24), "\n")
	if got := strings.TrimRight(lines[1], " "); len(lines) != 24 || got != "tab one" {
		t.Fatalf("%d lines, the second %q", len(lines), got)
	}
	if !strings.HasPrefix(lines[2], "tab two") || !strings.Contains(lines[3], "Steps") {
		t.Errorf("lines 3 and 4: %q, %q", lines[2], lines[3])
	}
	if got := layout.PaneSizes(80, 24)[0].Height; got != 18 {
		t.Errorf("a pane holds %d lines, want 18 with the header's two taken", got)
	}
}

func TestRenderDropsTheStatusThatDoesNotFit(t *testing.T) {
	got := stripped(twoPanes().Render(NewTheme(noColor("1")), 18, 24))
	if line, _, _ := strings.Cut(got, "\n"); line != " ws server apply  " {
		t.Errorf("title %q", line)
	}
}

// TestRenderFillsAnyWindow keeps a resize from breaking the screen. A window gets its exact height
// and no line past its width, around each size where the layout changes, and in tiny ones.
func TestRenderFillsAnyWindow(t *testing.T) {
	theme := NewTheme(noColor(""))
	long := strings.Repeat("0123456789 ", 30) + "\n" + strings.Repeat("line\n", 80) + "日本語\ttab"
	wide := twoPanes()
	wide.Title = strings.Repeat("title", 30)
	wide.Panes = append(wide.Panes, Pane{Title: wide.Title, Body: long, Weight: 9})
	wide.Footer = testFooter().Enable(1, false)
	widths := []int{1, 2, 3, 4, 5, 6, 8, 12, 23, 24, 47, 48, 71, 72, 73, 80, 81, 200}
	heights := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 24, 50}
	for _, layout := range []Layout{twoPanes(), wide, {}, {Panes: []Pane{{Body: long}}}} {
		for _, width := range widths {
			for _, height := range heights {
				got := layout.Render(theme, width, height)
				if lines := strings.Count(got, "\n") + 1; lines != height {
					t.Fatalf("%dx%d: %d lines", width, height, lines)
				}
				if size := lipgloss.Width(got); size > width {
					t.Fatalf("%dx%d: %d columns", width, height, size)
				}
			}
		}
	}
}

func TestRenderWithNoWindowDrawsNothing(t *testing.T) {
	if got := twoPanes().Render(NewTheme(noColor("")), 0, 0); got != "" {
		t.Errorf("render = %q", got)
	}
}

func TestViewIsAFullScreenWithNoMouse(t *testing.T) {
	view := twoPanes().View(NewTheme(noColor("")), 80, 24)
	if !view.AltScreen {
		t.Error("the view stays on the normal screen")
	}
	if view.MouseMode != tea.MouseModeNone {
		t.Errorf("mouse mode %v", view.MouseMode)
	}
}

func TestWhatStandsBelowThePanesTakesTheirRoom(t *testing.T) {
	layout := twoPanes()
	layout.Below = "first\nsecond"
	layout.Footer = NewFooter(Keys.Quit)
	if got, want := layout.PaneSizes(80, 24)[0].Height, 24-1-2-1-2; got != want {
		t.Errorf("%d lines in a pane, want %d", got, want)
	}
	lines := strings.Split(stripped(layout.Render(testTheme, 80, 24)), "\n")
	if got := lines[21]; !strings.HasPrefix(got, "first") {
		t.Errorf("line 21 = %q in:\n%s", got, strings.Join(lines, "\n"))
	}
	if got := lines[23]; !strings.Contains(got, "quit") {
		t.Errorf("last line = %q", got)
	}
}
