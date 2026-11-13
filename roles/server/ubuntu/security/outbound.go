package security

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/file"
	"github.com/zadykian/ws/engine/core/host/systemd"
)

// unitFile makes the file of a unit that the fill or the table needs.
func unitFile(m effect.Machine, name string) (file.File, error) {
	content, err := render(name)
	return file.File{M: m, Path: "/etc/systemd/system/" + name, Content: content, Mode: 0o644}, err
}

// outbound keeps what ws starts to the internet and ikeward0, in a table of ws's own that
// security-outbound.service loads. The fill, its units and its watchers go in before the table and
// its unit. So a run cut short, then a boot, finds the old table, or the new one with what fills
// its sets. The path unit watches from before the table loads (WantedBy=sysinit.target). It reads
// its unit first, as no restart has run yet. nft -c checks the table in the kernel, which a
// container without NET_ADMIN refuses. So it checks with systemd booted alone; CI's playbook job
// runs it on the file after the run (docs/checks.md).
func outbound(m effect.Machine, u effect.Units, out effect.Out) ([]host.Resource, error) {
	script, err := files.ReadFile("files/resolvers.sh")
	if err != nil {
		return nil, err
	}
	var fill []host.Resource
	for _, n := range []string{pathUnit, timerUnit, fillUnit} {
		f, err := unitFile(m, n)
		if err != nil {
			return nil, err
		}
		fill = append(fill, f)
	}
	table, err := render("outbound.nft")
	if err != nil {
		return nil, err
	}
	var nft host.Resource = file.File{M: m, Path: tableFile, Content: table, Mode: 0o644}
	if systemd.Booted(m) {
		nft = file.Validated{File: nft.(file.File), Check: func(p string) (string, []string) {
			return "/usr/sbin/nft", []string{"-c", "-f", p}
		}}
	}
	service, err := unitFile(m, outboundUnit)
	if err != nil {
		return nil, err
	}
	fillScript := file.File{M: m, Path: scriptFile, Content: script, Mode: 0o755}
	d := drift{m, out}
	return append(append([]host.Resource{
		file.Dir{M: m, Path: "/usr/local/lib/ws", Mode: 0o755},
		file.Dir{M: m, Path: tableDir, Mode: 0o755},
		fillScript,
	}, fill...), systemd.DaemonReload{M: m, U: u, Watch: fill},
		systemd.Unit{M: m, U: u, Out: out, Name: pathUnit, Enabled: true, Run: systemd.Running, Watch: fill},
		systemd.Unit{M: m, U: u, Out: out, Name: timerUnit, Enabled: true, Run: systemd.Running, Watch: fill},
		systemd.Unit{M: m, U: u, Out: out, Name: fillUnit, Run: systemd.Triggered, Watch: append(fill[:len(fill):len(fill)], d)},
		nft,
		service,
		systemd.DaemonReload{M: m, U: u, Watch: []host.Resource{service}},
		d,
		systemd.Unit{M: m, U: u, Out: out, Name: outboundUnit, Enabled: true, Run: systemd.Running, Reload: true,
			Watch: []host.Resource{fillScript, nft, service, d}},
	), nil
}
