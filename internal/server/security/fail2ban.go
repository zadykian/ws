package security

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// fail2ban bans addresses that fail to log in over SSH, starts now and at boot, and reloads
// where its jail changed.
func fail2ban(m effect.Machine, u effect.Units, out effect.Out) ([]host.Resource, error) {
	jail, err := files.ReadFile("files/sshd.local")
	if err != nil {
		return nil, err
	}
	file := host.File{M: m, Path: "/etc/fail2ban/jail.d/sshd.local", Content: jail, Mode: 0o644}
	return []host.Resource{
		file,
		host.Unit{M: m, U: u, Out: out, Name: "fail2ban.service", Enabled: true, Run: host.Running,
			Reload: true, Watch: []host.Resource{file}},
	}, nil
}
