package values

import "fmt"

// Sibling is a server that another server's values are checked against for a clash.
type Sibling struct {
	Name, File string
	Conf       Conf
}

// BadFileHint is the way out of a server's file that ws refuses.
func BadFileHint(name string) string {
	return "fix its file, or ws macos remove " + name
}

// RefuseSibling words the refusal of the server called name, whose file is unreadable or refused
// for cause. A run and a dry run refuse in these words.
func RefuseSibling(name string, cause error) error {
	return fmt.Errorf("server %s: %w: %s", name, cause, BadFileHint(name))
}
