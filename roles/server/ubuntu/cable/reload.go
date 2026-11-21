package cable

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/server/ubuntu/cable/internal/netplan"
	"github.com/zadykian/ws/roles/server/values"
)

// inForce is the two commands that put netplan's files in force. netplan writes networkd's files
// from its own. Then networkd's reload sets up each link whose file changed. networkd's reload alone
// would read the old files.
func inForce(m effect.Machine) []resource.Change {
	return []resource.Change{command{m: m, why: "write networkd's files from netplan's", name: "netplan", args: []string{"generate"}},
		command{m: m, why: "reload networkd, which sets up again each link whose file changed", name: "networkctl", args: []string{"reload"}}}
}

// Reload puts netplan's files in force where the cable's file changed since the last reload, as
// inForceMark tells. So a machine whose cable file stays, as one without the cable, never reloads.
// A netplan file edited by hand goes in force by netplan apply alone, not by ws. Cable is the
// cable of the values, and U the service manager's units, for Check.
type Reload struct {
	M     effect.Machine
	U     effect.Units
	Cable values.Cable
}

// inForceMark, in markDir, holds the SHA-256 of the cable's file that the last reload put in
// force. A machine where no reload put one in force, as where the file has never been, or went,
// has no mark. A failed reload leaves the mark unchanged, so the reload stays due.
const (
	markDir     = "/var/lib/ws"
	inForceMark = markDir + "/cable-in-force"
)

// Plan plans inForceMark's change after inForce's commands, where the file's SHA-256 differs from
// the mark's. Before a reload it checks the live cable file, as the file unit checks a new one.
// Without it, a netplan file added since that takes the port would go in force with the reload.
func (r Reload) Plan(ctx context.Context) ([]resource.Change, error) {
	sum, err := sumOf(r.M)
	if err != nil {
		return nil, err
	}
	mark, err := r.M.ReadRootFile(inForceMark)
	if errors.Is(err, fs.ErrNotExist) {
		mark, err = nil, nil
	}
	if err != nil || strings.TrimSpace(string(mark)) == sum {
		return nil, err
	}
	if err := r.cableHolds(ctx); err != nil {
		return nil, err
	}
	return append(inForce(r.M), marked{m: r.M, sum: sum}), nil
}

// sumOf gives the SHA-256 of the cable's file, read as root, or "" without the file.
func sumOf(m effect.Machine) (string, error) {
	file, err := m.ReadRootFile(netplan.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", sha256.Sum256(file)), nil
}

// cableHolds runs Check on the live cable file, where the port's interface and the file exist: the
// file unit is off without the interface, and absent without a port.
func (r Reload) cableHolds(ctx context.Context) error {
	if r.Cable.Port == "" {
		return nil
	}
	for _, path := range []string{"/sys/class/net/" + r.Cable.Port, netplan.Path} {
		if _, err := r.M.Stat(path); errors.Is(err, fs.ErrNotExist) {
			return nil
		} else if err != nil {
			return err
		}
	}
	return Check(ctx, r.M, r.U, r.Cable, netplan.Path)
}

// command is one of inForce's commands.
type command struct {
	m    effect.Machine
	why  string
	name string
	args []string
}

func (c command) String() string { return c.why }

func (c command) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, c.name, c.args...)
	return err
}

// marked writes sum to inForceMark, or removes the mark where sum is empty, as the file went.
type marked struct {
	m   effect.Machine
	sum string
}

func (c marked) String() string { return "note the cable's file that is in force" }

func (c marked) Apply(ctx context.Context) error {
	if c.sum == "" {
		_, err := c.m.RunRoot(ctx, "rm", "-f", "--", inForceMark)
		return err
	}
	if _, err := c.m.RunRoot(ctx, "mkdir", "-p", "--", markDir); err != nil {
		return err
	}
	return c.m.WriteRootFile(inForceMark, []byte(c.sum+"\n"), 0o600)
}
