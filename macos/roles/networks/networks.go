// Package networks sets up the routes and resolvers of the networks that the tunnel reaches.
package networks

import (
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

const resolverDir = "/etc/resolver"

// ownsFile is whether the file is missing or starts with ws's header.
func ownsFile(m effect.Machine, file string) (bool, error) {
	have, err := m.ReadRootFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	first, _, _ := strings.Cut(string(have), "\n")
	return first == resolverHeader, nil
}

// removal is the removal of a resolver that no server's file names, of kind Removal.
type removal struct{ resource.Change }

// Kind is removal.
func (removal) Kind() step.ChangeKind { return step.Removal }
