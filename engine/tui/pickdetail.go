package tui

import (
	"strings"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// detail draws notes, from the pane's top, in size. PgUp and PgDn move the list, so that what runs
// past the pane stays for `c`, which prints a step's changes whole.
func (s listScreen) detail(size widget.Size, notes []widget.Line) string {
	pane := widget.OutputPane{Theme: s.theme, Size: size}
	for _, line := range notes {
		if len(pane.Rows) >= size.Height {
			break
		}
		pane = pane.Add(line)
	}
	return pane.Move(len(pane.Rows)).View()
}

// warnings says why the scan stopped short, why a machine reported nothing, and which roles hold
// changes left out.
func (s listScreen) warnings(unread string, held []string) []widget.Line {
	var notes []widget.Line
	if s.problem != "" {
		notes = append(notes, widget.Line{Kind: widget.Warned, Text: "The scan stopped: " + s.problem})
	}
	if unread != "" {
		notes = append(notes, widget.Line{Kind: widget.Warned, Text: "Not scanned: " + unread})
	}
	if len(held) > 0 {
		text := "warning: changes left out in " + strings.Join(held, ", ") +
			", ahead of a step chosen. It can fail without them."
		notes = append(notes, widget.Line{Kind: widget.Warned, Text: text})
	}
	return notes
}

// notes is the words of the pane: what holds for the whole list, then the line under the cursor.
func (s pickScreen) notes() []widget.Line {
	notes := s.warnings("", s.list.passedOver())
	line, ok := s.list.current(s.at)
	if !ok || line.heading {
		return notes
	}
	row := s.list.rows[line.row]
	return append(notes, row.notes()...)
}

// notes says why the step stands as it does, then what the scan printed of it.
func (r pickRow) notes() []widget.Line {
	var notes []widget.Line
	if len(r.lockedBy) > 0 {
		notes = append(notes, widget.Line{Kind: widget.Said, Text: "Locked: " + strings.Join(r.lockedBy, ", ") + " waits on it."})
	}
	if r.waitsOn != "" {
		notes = append(notes, widget.Line{Kind: widget.Said, Text: "Waits on " + r.waitsOn + "."})
	}
	return append(append(notes, widget.Line{Kind: widget.Said, Text: r.state.words(r.after)}), r.said...)
}

// words says how the scan left a step. after names the step whose changes it stands behind.
func (s stepState) words(after string) string {
	switch {
	case after != "":
		return "Not planned: it follows the changes of " + after + "."
	case s == stepChanged:
		return "Holds changes."
	case s == stepSame:
		return "Nothing to change."
	case s == stepFailed:
		return "The scan failed here."
	case s == stepSkipped:
		return "Not scanned: the scan ended first."
	}
	return "Not scanned yet."
}
