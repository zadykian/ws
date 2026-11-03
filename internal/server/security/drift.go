package security

import (
	"bytes"
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// drift finds the outbound table in force other than its file's. A run cut short before the
// unit's reload leaves the old table in force, and a table flushed or deleted by hand leaves the
// unit active. Nothing else would reload the unit, so the plan compares the kernel's table with
// the file's, loaded in a network namespace of its own as the unit loads it. -s leaves out
// counters' values, not the counters nor the sets' elements. unshare needs CAP_SYS_ADMIN, which
// CI's container lacks, so it skips without systemd. A table that fails to load, or no table in
// force, counts as a difference: the reload shows which. A file or fill not yet written has none
// to load, and its write is a change of its own.
type drift struct {
	m   effect.Machine
	out effect.Out
}

type reloadDue struct{}

func (reloadDue) String() string {
	return "reload the outbound table: the one in force is not the file's"
}

func (reloadDue) Apply(context.Context) error { return nil }

// list is the command that prints the table that the file loads, in a network namespace of its own.
var list = []string{"--net", "sh", "-c", `"$1" "$2" && nft -s list table inet ws`, "sh", scriptFile, tableFile}

// Plan plans a reload where the tables differ.
func (d drift) Plan(ctx context.Context) ([]host.Change, error) {
	if !host.Booted(d.m) {
		d.out.Say("outbound table: not compared with the kernel's, systemd is not running")
		return nil, nil
	}
	for _, p := range []string{scriptFile, tableFile} {
		_, err := d.m.RootAttrs(p)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil // written as a change of its own
		}
		if err != nil {
			return nil, err
		}
	}
	loaded, err := d.m.RunRoot(ctx, "unshare", list...)
	if err != nil {
		return []host.Change{reloadDue{}}, nil //nolint:nilerr // a table that fails to load is a reload due
	}
	inForce, err := d.m.RunRoot(ctx, "nft", "-s", "list", "table", "inet", "ws")
	if err != nil || !bytes.Equal(inForce, loaded) {
		return []host.Change{reloadDue{}}, nil //nolint:nilerr // no table in force is a reload due
	}
	return nil, nil
}
