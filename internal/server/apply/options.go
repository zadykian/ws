// Package apply runs the server's roles for ws server apply: each as Go steps or, in runs of
// consecutive roles, by one ansible-playbook.
package apply

import (
	"errors"
	"fmt"
	"strings"
)

// Options are the arguments of ws server apply.
type Options struct {
	// Check plans the steps and passes --check --diff to the playbook.
	Check bool
	// Only names the roles to run, or is empty for all.
	Only []string
	// Args go to ansible-playbook, as the arguments after "--".
	Args []string
}

// Parse reads the arguments of ws server apply: [--check] [--only NAME,...] [-- ARG...].
func Parse(args []string) (Options, error) {
	var o Options
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			o.Args = args[i+1:]
			return o, nil
		case a == "--check":
			o.Check = true
		case a == "--only" || strings.HasPrefix(a, "--only="):
			list, ok := strings.CutPrefix(a, "--only=")
			if !ok {
				if i++; i == len(args) {
					return o, errors.New("--only needs role names")
				}
				list = args[i]
			}
			for name := range strings.SplitSeq(list, ",") {
				if name == "" {
					return o, fmt.Errorf("--only %q holds an empty role name", list)
				}
				o.Only = append(o.Only, name)
			}
		default:
			return o, fmt.Errorf("unknown argument %q", a)
		}
	}
	return o, nil
}
