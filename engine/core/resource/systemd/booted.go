// Package systemd runs units and user units, reloads systemd and enables lingering.
package systemd

import "github.com/zadykian/ws/engine/core/effect"

// Booted reports whether systemd booted the machine, by sd_booted(3)'s test: the directory
// /run/systemd/system exists. In CI's container it does not, so a step that needs systemd skips.
func Booted(m effect.Machine) bool {
	info, err := m.Stat("/run/systemd/system")
	return err == nil && info.Mode.IsDir()
}
