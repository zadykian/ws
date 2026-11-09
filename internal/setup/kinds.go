package setup

import "github.com/zadykian/ws/step"

// The rest of the changes stay other: each is a line for a person, a key made, or a file kept
// beside a write.

// Kind is write.
func (Write) Kind() step.ChangeKind { return step.Write }

// Kind is write.
func (MakeDir) Kind() step.ChangeKind { return step.Write }

// Kind is write.
func (SetMode) Kind() step.ChangeKind { return step.Write }

// Kind is write.
func (KeyCopy) Kind() step.ChangeKind { return step.Write }

// Kind is install.
func (Install) Kind() step.ChangeKind { return step.Install }

func (cldInstall) Kind() step.ChangeKind { return step.Install }
func (cldUpdate) Kind() step.ChangeKind  { return step.Install }

// Kind is restart.
func (Launchctl) Kind() step.ChangeKind { return step.Restart }

// Kind says install for brew, the one package manager of the Mac's runs.
func (r Run) Kind() step.ChangeKind {
	if r.Name == "brew" {
		return step.Install
	}
	return step.OtherChange
}
