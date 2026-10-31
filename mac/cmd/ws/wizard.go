package main

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"strings"
)

const wizardUsage = "usage: ws wizard [--check] [--update]"

// wizardEnv is what ws wizard takes from the process, so that a test gives it another.
type wizardEnv struct {
	goos   string
	uid    int
	getenv func(string) string
	stderr io.Writer
}

type wizardOptions struct{ check, update bool }

// wizardOptionsOf reads the arguments, or says why it can't. Help is no error.
func wizardOptionsOf(args []string) (o wizardOptions, help bool, err error) {
	for _, a := range args {
		switch a {
		case "--check":
			o.check = true
		case "--update":
			o.update = true
		case "-h", "--help":
			return o, true, nil
		default:
			return o, false, fmt.Errorf("unknown argument '%s'; %s", a, wizardUsage)
		}
	}
	return o, false, nil
}

// wizardRefusal returns why ws wizard does not run here, or an empty string. It names the system
// as uname -s does.
func wizardRefusal(e wizardEnv) string {
	switch {
	case e.goos != "darwin":
		return "it sets up a Mac, not " + strings.ToUpper(e.goos[:1]) + e.goos[1:]
	case e.uid == 0:
		return "run it as yourself, not root: it asks sudo where it needs root"
	}
	return ""
}

// wizardCommand runs ws wizard.
func wizardCommand(args []string, e wizardEnv) int {
	prog := cmp.Or(e.getenv("WS_PROG"), "ws wizard")
	_, help, err := wizardOptionsOf(args)
	if help {
		_, _ = fmt.Fprintln(e.stderr, strings.Replace(wizardUsage, "ws wizard", prog, 1))
		return 0
	}
	if err == nil && wizardRefusal(e) != "" {
		err = errors.New(wizardRefusal(e))
	}
	if err != nil {
		_, _ = fmt.Fprintf(e.stderr, "%s: %v\n", prog, err)
		return 1
	}
	return 0
}
