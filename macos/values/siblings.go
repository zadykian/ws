package values

import "fmt"

// Sibling is a server that another server's values are checked against for a clash.
type Sibling struct {
	Name, File string
	Conf       Conf
}

// BadFileHint is the way out of a server's file that ws refuses: fixing it, or removing the server.
// Where the server has guests, remove refuses its removal and theirs while its file stays refused,
// so fixing the file is the one way.
func BadFileHint(name string, guests []string) string {
	if len(guests) > 0 {
		return "fix its file"
	}
	return "fix its file, or ws macos remove " + name
}

// RefuseSibling words the refusal of the server called name, whose file is unreadable or refused
// for cause, and which hosts guests. A run and a dry run refuse in these words.
func RefuseSibling(name string, cause error, guests []string) error {
	return fmt.Errorf("server %s: %w: %s", name, cause, BadFileHint(name, guests))
}
