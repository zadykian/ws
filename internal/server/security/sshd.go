package security

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// sshd allows SSH by key only, and keeps the tunnel's peers clear of its penalties. sshd -G parses the file, as a config of its own, before it is in
// place: sshd refuses to start with a bad one, and a reboot would then lock SSH out. Unlike
// sshd -t, it needs no /run/sshd, which exists only once ssh.service has started since boot.
// ssh.service tests the config again before it reloads. ssh.socket starts sshd at the first
// login; before that, a change starts it. The reload comes in this step, not at the end of the
// run: if a later step fails, the next run has no change to reload for, and a running sshd would
// go on taking passwords.
func sshd(m effect.Machine, u effect.Units, out effect.Out) ([]host.Resource, error) {
	keys, err := files.ReadFile("files/10-keys-only.conf")
	if err != nil {
		return nil, err
	}
	peers, err := files.ReadFile("files/10-tunnel-peers.conf")
	if err != nil {
		return nil, err
	}
	check := func(path string) (string, []string) {
		return "/usr/sbin/sshd", []string{"-G", "-f", path}
	}
	dropIns := []host.Resource{
		host.Validated{
			File:  host.File{M: m, Path: "/etc/ssh/sshd_config.d/10-keys-only.conf", Content: keys, Mode: 0o644},
			Check: check,
		},
		host.Validated{
			File:  host.File{M: m, Path: "/etc/ssh/sshd_config.d/10-tunnel-peers.conf", Content: peers, Mode: 0o644},
			Check: check,
		},
	}
	return append(dropIns,
		host.Unit{M: m, U: u, Out: out, Name: "ssh.service", Run: host.Running, Reload: true, Watch: dropIns}), nil
}
