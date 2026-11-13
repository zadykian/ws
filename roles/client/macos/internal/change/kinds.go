package change

import "github.com/zadykian/ws/engine/core/step"

// The rest of the changes stay other: each is a line for a person.

// Kind says install for brew, the one package manager of the Mac's runs.
func (r Run) Kind() step.ChangeKind {
	if r.Name == "brew" {
		return step.Install
	}
	return step.OtherChange
}
