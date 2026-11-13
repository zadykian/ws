package base

import (
	_ "embed"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

//go:embed files/20auto-upgrades
var autoUpgrades []byte

// upgradesParts turns on unattended upgrades with the settings the package writes when its debconf
// question is answered yes, Ubuntu's default. The stock 50unattended-upgrades, left unchanged,
// allows Ubuntu's security updates. devtools' 52unattended-upgrades-github-cli adds gh's releases. docs/base.md says what ucf does with the file's header at
// an upgrade. Then apt's daily timers and unattended-upgrades' shutdown check are enabled and
// started; a machine without a booted systemd skips them.
func upgradesParts(m effect.Machine, u effect.Units, out effect.Out) []resource.Resource {
	parts := []resource.Resource{file.File{M: m, Path: "/etc/apt/apt.conf.d/20auto-upgrades", Content: autoUpgrades, Mode: 0o644}}
	for _, name := range []string{"apt-daily.timer", "apt-daily-upgrade.timer", "unattended-upgrades.service"} {
		parts = append(parts, systemd.Unit{M: m, U: u, Out: out, Name: name, Enabled: true, Run: systemd.Running})
	}
	return parts
}
