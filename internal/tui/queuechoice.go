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

// due says that a run has work in r: the scan found some, or never reached it. What no scan named
// under a stage that found nothing to do has none.
func (l pickList) due(r pickRow) bool {
	switch r.state {
	case stepSame, stepLeftOut:
		return false
	case stepOpen:
		return !slices.ContainsFunc(l.rows, func(o pickRow) bool { return o.number > 0 && o.state == stepSame && under(o)(r) })
	}
	return true
}

// opened returns the queue as a scan's end leaves it. What is due starts chosen, as the wizard left
// alone runs each stage it reaches, and a check stops short on a new Mac.
func (l pickList) opened() pickList {
	due := l.due
	l.rows = slices.Clone(l.rows)
	for i, r := range l.rows {
		l.rows[i].chosen = due(r)
	}
	return l.settled().whole(due)
}

// whole returns the list with every step on in a stage that want takes and that has none on: ws's
// run can be due while its roles read clean, and a stage with nothing on wouldn't run.
func (l pickList) whole(want func(pickRow) bool) pickList {
	l.rows = slices.Clone(l.rows)
	for _, r := range l.rows {
		if on, _ := l.count(under(r)); r.number == 0 || on > 0 || !want(r) {
			continue
		}
		for i := range l.rows {
			if under(r)(l.rows[i]) {
				l.rows[i].chosen = true
			}
		}
	}
	return l.settled()
}
