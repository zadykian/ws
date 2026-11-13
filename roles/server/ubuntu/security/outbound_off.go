package security

import (
	"context"
	"errors"
	"io/fs"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/file"
	"github.com/zadykian/ws/engine/core/host/systemd"
)

// unitDirs are where a link can enable a unit: /etc for the units, /run for those enabled with
// --runtime.
var unitDirs = []string{"/etc/systemd/system", "/run/systemd/system"}

// fillRecord is the fill's record of what it set.
const fillRecord = "/run/security-resolvers.conf"

// outboundUnits are the table's units in the order they stop: a unit that starts another, as a
// .path unit, before it.
var outboundUnits = []string{pathUnit, timerUnit, fillUnit, outboundUnit}

// outboundOff takes the outbound table off, as security_outbound: false wants. The stop of
// security-outbound.service removes its table, but not after systemd reloads without the unit's
// file, which drops ExecStop. So the units stop first, by D-Bus, which sees a unit whose file is
// gone, active or failed; a failed one is reset. Then the links that enable them go, in any
// target's .wants or .requires, with their files, and systemd reads its units again. A table no
// stop removed outlives them, as one loaded by hand with nft -f.
func outboundOff(m effect.Machine, u effect.Units, out effect.Out) []host.Resource {
	var rs []host.Resource
	for _, n := range outboundUnits {
		rs = append(rs, systemd.Unit{M: m, U: u, Out: out, Name: n, Run: systemd.Stopped})
	}
	gone := []host.Resource{links{m}}
	for _, n := range outboundUnits {
		gone = append(gone, file.Absent{M: m, Path: "/etc/systemd/system/" + n})
	}
	gone = append(gone, file.Absent{M: m, Path: tableDir}, file.Absent{M: m, Path: fillRecord})
	return append(append(rs, gone...), systemd.DaemonReload{M: m, U: u, Watch: gone}, table{m, out})
}

// links is the removal of the links that enable the outbound table's units, in a directory of
// systemd's or a directory under it.
type links struct{ m effect.Machine }

type removeLink struct {
	m effect.Machine
	p string
}

func (c removeLink) String() string { return "remove " + c.p }

func (c removeLink) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "rm", "-f", "--", c.p)
	return err
}

// Plan plans the removal of each link that is named for a unit.
func (l links) Plan(context.Context) ([]host.Change, error) {
	var cs []host.Change
	for _, root := range unitDirs {
		names, err := l.m.ReadDir(root)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		dirs := []string{root}
		for _, n := range names {
			if info, err := l.m.Stat(path.Join(root, n)); err == nil && info.Link == "" && info.Mode.IsDir() {
				dirs = append(dirs, path.Join(root, n))
			}
		}
		for _, d := range dirs {
			for _, n := range outboundUnits {
				p := path.Join(d, n)
				info, err := l.m.Stat(p)
				if err != nil || info.Link == "" {
					continue // absent, or the unit's own file
				}
				cs = append(cs, removeLink{l.m, p})
			}
		}
	}
	return cs, nil
}

// table removes the table of ws's from the kernel where one is loaded. A dry run, which stops no
// unit, finds the table and shows its removal. Without systemd it plans nothing.
type table struct {
	m   effect.Machine
	out effect.Out
}

type destroy struct{ m effect.Machine }

func (destroy) String() string { return "remove the outbound table" }

func (c destroy) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "nft", "destroy", "table", "inet", "ws")
	return err
}

// Plan plans the removal where the kernel holds the table. No nft, or no table, is none to remove.
func (t table) Plan(ctx context.Context) ([]host.Change, error) {
	if !systemd.Booted(t.m) {
		t.out.Say("outbound table: not looked for, systemd is not running")
		return nil, nil
	}
	if _, err := t.m.RunRoot(ctx, "nft", "list", "table", "inet", "ws"); err != nil {
		return nil, nil //nolint:nilerr // a failed list is no table in force
	}
	return []host.Change{destroy{t.m}}, nil
}
