package tui

import "strings"

// detail draws what the pane says, from its top, in size. PgUp and PgDn move the list, so that
// what runs past the pane stays for `c`, which prints a step's changes whole.
func (s pickScreen) detail(size Size) string {
	pane := outputPane{theme: s.theme, size: size}
	for _, line := range s.notes() {
		if len(pane.rows) >= size.Height {
			break
		}
		pane = pane.add(line)
	}
	return pane.move(len(pane.rows)).view()
}

// notes is the words of the pane, for the line under the cursor.
func (s pickScreen) notes() []Line {
	var notes []Line
	line, ok := s.list.current()
	if !ok || line.heading {
		return notes
	}
	return append(notes, s.list.rows[line.row].notes()...)
}

// notes says why the step stands as it does, then what the scan printed of it.
func (r pickRow) notes() []Line {
	var notes []Line
	if len(r.lockedBy) > 0 {
		notes = append(notes, Line{Said, "Locked: " + strings.Join(r.lockedBy, ", ") + " waits on it."})
	}
	if r.waitsOn != "" {
		notes = append(notes, Line{Said, "Waits on " + r.waitsOn + "."})
	}
	return append(append(notes, Line{Said, r.state.words(r.after)}), r.said...)
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
