package tui

import "slices"

// under returns the test of a step that the stage r runs.
func under(r pickRow) func(pickRow) bool {
	return func(o pickRow) bool { return o.stage == r.name }
}

// nests says that r is a stage with steps of its own to choose.
func (l pickList) nests(r pickRow) bool {
	return r.number > 0 && slices.ContainsFunc(l.rows, under(r))
}

// tickStages ticks a stage by the steps it runs, in rows the caller owns.
func (l pickList) tickStages() {
	for i, r := range l.rows {
		if l.nests(r) {
			l.rows[i].chosen = slices.ContainsFunc(l.rows, func(o pickRow) bool { return under(r)(o) && o.chosen })
		}
	}
}

// holdStages runs a locked stage whole, in rows the caller owns: a partial run leaves a stage
// pending, so the stage that waits on it would only skip.
func (l pickList) holdStages() {
	for _, r := range l.rows {
		if r.number == 0 || len(r.lockedBy) == 0 {
			continue
		}
		for i := range l.rows {
			if under(r)(l.rows[i]) {
				l.rows[i].lockedBy = r.lockedBy
			}
		}
	}
}

// box draws r's checkbox. A stage's says "[-]" where it runs some of its steps.
func (l pickList) box(r pickRow) string {
	if r.number > 0 {
		if on, all := l.count(under(r)); on > 0 && on < all {
			return "[-]"
		}
	}
	return r.box()
}

// partial returns what each stage that runs in part runs, by Options.Only's names.
func (l pickList) partial() map[string][]string {
	var parts map[string][]string
	for _, r := range l.rows {
		if r.number == 0 {
			continue
		}
		if on, all := l.count(under(r)); on > 0 && on < all {
			if parts == nil {
				parts = map[string][]string{}
			}
			parts[r.name] = l.entries(under(r))
		}
	}
	return parts
}
