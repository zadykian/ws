package security

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// fail2ban bans addresses that fail to log in over SSH, starts now and at boot, and reloads
// where its jail changed.
func fail2ban(m effect.Machine, u effect.Units, out effect.Out) ([]resource.Resource, error) {
	jail, err := files.ReadFile("files/sshd.local")
	if err != nil {
		return nil, err
	}
	jailFile := file.File{M: m, Path: "/etc/fail2ban/jail.d/sshd.local", Content: jail, Mode: 0o644}
	return []resource.Resource{
		jailFile,
		systemd.Unit{M: m, U: u, Out: out, Name: "fail2ban.service", Enabled: true, Run: systemd.Running,
			Reload: true, Watch: []resource.Resource{jailFile}},
	}, nil
}
