package setup

import "strings"

// Run runs a command, through sudo where Root is set.
type Run struct {
	Root bool
	Name string
	Args []string
	// Label is how --check and the run's lines show the command, where it differs from Name and
	// Args: a program's full path, which sudo needs, is long, and a name with spaces is quoted.
	Label string
}

// String is the line that --check prints.
func (r Run) String() string {
	return "would run " + r.line()
}

func (r Run) line() string {
	if r.Label != "" {
		return r.Label
	}
	return strings.Join(append([]string{r.Name}, r.Args...), " ")
}
