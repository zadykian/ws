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
	// Secrets is the age-encrypted file of --secrets, or empty.
	Secrets string
	// Args go to ansible-playbook, as the arguments after "--".
	Args []string
}

// Parse reads the arguments of ws server apply: [--check] [--only NAME,...] [--secrets FILE] [-- ARG...].
func Parse(args []string) (Options, error) {
	var o Options
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			o.Args = args[i+1:]
			return o, refuseArgs(o.Args)
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

// refuseArgs refuses what ws sets itself, or what --only replaces, among the playbook's arguments.
func refuseArgs(args []string) error {
	for i, a := range args {
		switch {
		case isVaultFlag(a):
			return fmt.Errorf("%s after --: use ws server apply --secrets with an age file", a)
		case a == "-e" || a == "--extra-vars":
			if i+1 < len(args) && strings.HasPrefix(args[i+1], "@") {
				return fmt.Errorf("%s %s after --: use ws server apply --secrets with an age file", a, args[i+1])
			}
		case strings.HasPrefix(a, "-e@") || strings.HasPrefix(a, "--extra-vars=@"):
			return fmt.Errorf("%s after --: use ws server apply --secrets with an age file", a)
		case a == "-C" || a == "--check":
			return fmt.Errorf("%s after --: use ws server apply --check", a)
		case a == "-t" || a == "--tags" || strings.HasPrefix(a, "--tags=") ||
			(len(a) > 2 && a[0] == '-' && a[1] == 't'):
			return fmt.Errorf("%s after --: use ws server apply --only with role names", a)
		}
	}
	return nil
}

// ExtraVars returns the KEY=VALUE pairs that the arguments after -- set with -e or --extra-vars. A
// file (@FILE) or a JSON object holds none that can be read here.
func ExtraVars(args []string) []string {
	var pairs []string
	for i, a := range args {
		var val string
		switch {
		case (a == "-e" || a == "--extra-vars") && i+1 < len(args):
			val = args[i+1]
		case strings.HasPrefix(a, "--extra-vars="):
			val = strings.TrimPrefix(a, "--extra-vars=")
		case strings.HasPrefix(a, "-e") && len(a) > 2 && a != "--extra-vars":
			val = a[2:]
		}
		if !strings.HasPrefix(val, "@") && !strings.HasPrefix(val, "{") {
			pairs = append(pairs, strings.Fields(val)...)
		}
	}
	return pairs
}

// isVaultFlag reports whether the argument is one of ansible-playbook's for an Ansible Vault.
func isVaultFlag(a string) bool {
	for _, f := range []string{"--ask-vault-pass", "--ask-vault-password", "--vault-id", "--vault-pass", "--vault-password-file"} {
		if a == f || strings.HasPrefix(a, f+"=") {
			return true
		}
	}
	return a == "-J"
}
