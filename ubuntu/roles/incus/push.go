package incus

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// Pushed keeps a file of the host in a guest, as root's. It pushes the file where the guest's copy
// differs by its SHA-256, or is missing. The owner and the mode are set as the push writes them,
// and are not read again. A guest that does not run is not read, and the push is due.
type Pushed struct {
	M     effect.Machine
	Guest string
	// From is the file on the host, and To its path in the guest.
	From, To string
	// Mode is the file's mode in the guest, as 0755.
	Mode string
	// Optional leaves the guest's copy alone where the host holds no file.
	Optional bool
}

// Plan plans the push where the guest's copy differs.
func (p Pushed) Plan(ctx context.Context) ([]resource.Change, error) {
	want, err := p.M.ReadFile(p.From)
	if errors.Is(err, fs.ErrNotExist) && p.Optional {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	up, err := running(ctx, p.M, p.Guest)
	if err != nil || !up {
		return []resource.Change{push{p}}, err
	}
	out, err := execIn(ctx, p.M, probeTimeout, p.Guest, sumScript, p.To)
	if code, ok := exitCode(err); ok && code == missing {
		return []resource.Change{push{p}}, nil
	}
	if err != nil {
		return nil, fromGuest(err)
	}
	if have, _, _ := strings.Cut(string(out), " "); have != fmt.Sprintf("%x", sha256.Sum256(want)) {
		return []resource.Change{push{p}}, nil
	}
	return nil, nil
}

type push struct{ p Pushed }

func (c push) String() string {
	return fmt.Sprintf("push %s to %s in the guest %s, mode %s", c.p.From, c.p.To, c.p.Guest, c.p.Mode)
}

func (c push) Apply(ctx context.Context) error {
	_, err := c.p.M.Run(ctx, "incus", "file", "push", "--create-dirs", "--uid", "0", "--gid", "0", "--mode", c.p.Mode,
		c.p.From, c.p.Guest+c.p.To)
	return fromGuest(err)
}
