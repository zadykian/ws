package tui

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
	line, ok := s.list.current()
	if !ok || line.heading {
		return nil
	}
	return s.list.rows[line.row].said
}
