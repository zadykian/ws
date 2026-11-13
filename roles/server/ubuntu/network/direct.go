package network

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// direct is the cable from the Mac's dock to the server: the server's end gets Address from a
// netplan file of its own. With Interface empty, the file goes, and so does the address. A machine
// without the cable's port, as a cloud VM or CI's container, keeps its netplan files as they are.
type direct struct {
	M                  effect.Machine
	U                  effect.Units
	Out                effect.Out
	Interface, Address string
}

// readJSON runs a command and reads its standard output as JSON.
func (d direct) readJSON(ctx context.Context, into any, name string, args ...string) error {
	out, err := d.M.Run(ctx, name, args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(out, into); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}
