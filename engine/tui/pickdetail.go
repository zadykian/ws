package tui

import (
	"strings"

	"github.com/zadykian/ws/engine/tui/internal/widget"
)

// Detail draws notes, from the pane's top, in size. PgUp and PgDn move the list, so that what runs
// past the pane stays for `c`, which prints a step's changes whole.
func (s ListScreen) Detail(size widget.Size, notes []widget.Line) string {
	pane := widget.OutputPane{Theme: s.Theme, Size: size}
	for _, line := range notes {
		if len(pane.Rows) >= size.Height {
			break
		}
		pane = pane.Add(line)
	}
	return pane.Move(len(pane.Rows)).View()
}

// Warnings says why the scan stopped short, why a machine reported nothing, and which roles hold
// changes left out.
func (s ListScreen) Warnings(unread string, held []string) []widget.Line {
	var notes []widget.Line
	if s.Problem != "" {
		notes = append(notes, widget.Line{Kind: widget.Warned, Text: "The scan stopped: " + s.Problem})
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

// Notes says why the step stands as it does, then what the scan printed of it.
func (r PickRow) Notes() []widget.Line {
	var notes []widget.Line
	if len(r.LockedBy) > 0 {
		notes = append(notes, widget.Line{Kind: widget.Said, Text: "Locked: " + strings.Join(r.LockedBy, ", ") + " waits on it."})
	}
	if r.WaitsOn != "" {
		notes = append(notes, widget.Line{Kind: widget.Said, Text: "Waits on " + r.WaitsOn + "."})
	}
	return append(append(notes, widget.Line{Kind: widget.Said, Text: r.State.words(r.After)}), r.Said...)
}

// words says how the scan left a step. after names the step whose changes it stands behind.
func (s StepState) words(after string) string {
	switch {
	case after != "":
		return "Not planned: it follows the changes of " + after + "."
	case s == StepChanged:
		return "Holds changes."
	case s == StepSame:
		return "Nothing to change."
	case s == StepFailed:
		return "The scan failed here."
	case s == StepSkipped:
		return "Not scanned: the scan ended first."
	}
	return "Not scanned yet."
}
