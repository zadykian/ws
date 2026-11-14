package conf

import (
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
)

// Sibling is another server's file beside mac.conf, parsed by the server's scope.
type Sibling struct {
	Name, File string
	Conf       Conf
}

// Siblings returns the servers of servers/ beside mac.conf at mac, other than the one whose file is
// at server, sorted by name. It leaves out a file that is missing or refused: apply refuses such a
// file before its steps run. This is the one way to read the other servers.
func Siblings(m effect.Machine, mac, server string) ([]Sibling, error) {
	names, err := Servers(m, mac)
	if err != nil {
		return nil, err
	}
	var others []Sibling
	for _, n := range names {
		file := ServerPath(mac, n)
		if file == server {
			continue
		}
		c, err := load(m, file, ServerFile.Parse)
		if err != nil {
			continue
		}
		others = append(others, Sibling{Name: n, File: file, Conf: c})
	}
	return others, nil
}

// BadFileHint is the way out of a server's file that ws refuses.
func BadFileHint(name string) string {
	return "fix its file, or ws client macos remove " + name
}

// RefuseSibling words the refusal of the server called name, whose file is unreadable or refused
// for cause. A run and a dry run refuse in these words.
func RefuseSibling(name string, cause error) error {
	return fmt.Errorf("server %s: %w: %s", name, cause, BadFileHint(name))
}
