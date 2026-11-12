package security

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

// sshd allows SSH by key only. sshd -G parses the file, as a config of its own, before the file
// goes in place. sshd refuses to start with a bad one, and a reboot would then lock SSH out. Unlike
// sshd -t, it needs no /run/sshd, which exists only once ssh.service has started since boot.
// ssh.service tests the config again before it reloads. ssh.socket starts sshd at the first login;
// before that, a change starts it. The reload comes in this step, not at the end of the run. If a
// later step fails, the next run has no change to reload for, and a running sshd would go on taking
// passwords.
func sshd(m effect.Machine, u effect.Units, out effect.Out, user string) ([]host.Resource, error) {
	conf, err := host.Render(files, "files/10-keys-only.conf.tmpl", struct{ Root bool }{user == "root"})
	if err != nil {
		return nil, err
	}
	file := host.Validated{
		File: host.File{M: m, Path: "/etc/ssh/sshd_config.d/10-keys-only.conf", Content: conf, Mode: 0o644},
		Check: func(path string) (string, []string) {
			return "/usr/sbin/sshd", []string{"-G", "-f", path}
		},
	}
	return []host.Resource{
		file,
		host.Unit{M: m, U: u, Out: out, Name: "ssh.service", Run: host.Running, Reload: true,
			Watch: []host.Resource{file}},
	}, nil
}
