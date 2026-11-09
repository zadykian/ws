package base

import (
	_ "embed"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

//go:embed files/20auto-upgrades
var autoUpgrades []byte

// upgradesParts turns on unattended upgrades with the settings the package writes when its debconf
// question is answered yes, Ubuntu's default. The stock 50unattended-upgrades, left unchanged,
// allows Ubuntu's security updates alone. docs/base.md says what ucf does with the file's header at
// an upgrade. Then apt's daily timers and unattended-upgrades' shutdown check are enabled and
// started; a machine without a booted systemd skips them.
func upgradesParts(m effect.Machine, u effect.Units, out effect.Out) []host.Resource {
	parts := []host.Resource{host.File{M: m, Path: "/etc/apt/apt.conf.d/20auto-upgrades", Content: autoUpgrades, Mode: 0o644}}
	for _, name := range []string{"apt-daily.timer", "apt-daily-upgrade.timer", "unattended-upgrades.service"} {
		parts = append(parts, host.Unit{M: m, U: u, Out: out, Name: name, Enabled: true, Run: host.Running})
	}
	return parts
}
