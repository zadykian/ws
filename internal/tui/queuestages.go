package tui

// queueStage is a stage of a queue: its own row, the side that shows it, and a list of what it runs.
type queueStage struct {
	row  pickRow
	side Side
	list pickList
}

// queue is the stages in the order they run, each with a list of its own.
type queue struct {
	stages []queueStage
}

// queueOf returns the queue of stages. A group named as its stage has no heading, and one with no
// steps stands for a role no scan has read.
func queueOf(stages []Stage) queue {
	var q queue
	for i, s := range stages {
		groups := make([]listGroup, len(s.Nested))
		for j, g := range s.Nested {
			groups[j] = listGroup{Group: g}
			switch {
			case g.Name == s.Name:
				groups[j].form = groupBare
			case len(g.Steps) == 0:
				groups[j].form = groupUnread
			}
		}
		row := pickRow{stepRow: stepRow{index: i, name: s.Name}, waitsOn: s.Waits}
		q.stages = append(q.stages, queueStage{row: row, side: s.Side, list: newFormedList(groups)})
	}
	return q
}
