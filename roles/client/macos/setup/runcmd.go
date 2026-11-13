package setup

import "strings"

// Run runs a command, through sudo where Root is set.
type Run struct {
	Root bool
	Name string
	Args []string
	// Label is how --check and the run's lines show the command, where it differs from Name and
	// Args. A program's full path, which sudo needs, is long, and a name with spaces is quoted.
	Label string
}

// String is the line that --check prints.
func (r Run) String() string {
	return "would run " + r.Line()
}

// Line is how --check and a run's lines show the command.
func (r Run) Line() string {
	if r.Label != "" {
		return r.Label
	}
	return strings.Join(append([]string{r.Name}, r.Args...), " ")
}
