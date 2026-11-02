// Package apply runs the server's roles for ws server apply, each as Go steps.
package apply

import (
	"errors"
	"fmt"
	"strings"
)

// Options are the arguments of ws server apply.
type Options struct {
	// Check plans the steps and applies none.
	Check bool
	// Only names the roles to run, or is empty for all.
	Only []string
	// Secrets is the age-encrypted file of --secrets, or empty.
	Secrets string
}

// Parse reads the arguments of ws server apply: [--check] [--only NAME,...] [--secrets FILE].
func Parse(args []string) (Options, error) {
	var o Options
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--check":
			o.Check = true
		case a == "--secrets" || strings.HasPrefix(a, "--secrets="):
			file, ok := strings.CutPrefix(a, "--secrets=")
			if !ok {
				if i++; i == len(args) {
					return o, errors.New("--secrets needs a file")
				}
				file = args[i]
			}
			if file == "" || o.Secrets != "" {
				return o, errors.New("--secrets takes one file")
			}
			o.Secrets = file
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
