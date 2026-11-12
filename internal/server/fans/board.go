package fans

import (
	"context"
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
)

// boardVendorFile is where DMI gives the board's maker.
const boardVendorFile = "/sys/class/dmi/id/board_vendor"

// onBoard reports whether the machine's board is the vendor's. A machine with another board, as a
// cloud VM or CI's container, keeps its fans as they are. An empty vendor matches no machine.
func onBoard(m effect.Machine, vendor string) (bool, error) {
	if vendor == "" {
		return false, nil
	}
	text, err := m.ReadFile(boardVendorFile)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(text)) == vendor, nil
}

// board notes that the machine lacks the board whose fans the role drives.
type board struct {
	m      effect.Machine
	out    effect.Out
	vendor string
}

func (b board) Plan(context.Context) ([]host.Change, error) {
	ok, err := onBoard(b.m, b.vendor)
	if err != nil {
		return nil, err
	}
	if !ok {
		maker := b.vendor
		if maker == "" {
			maker = "no maker"
		}
		b.out.Say("fans: this machine's board is not one of %s's, so the role leaves its fans as they are", maker)
	}
	return nil, nil
}

// gated makes the parts only on the board whose fans the role drives.
func gated(m effect.Machine, vendor string, parts func(context.Context) ([]host.Resource, error)) func(context.Context) ([]host.Resource, error) {
	return func(ctx context.Context) ([]host.Resource, error) {
		ok, err := onBoard(m, vendor)
		if err != nil || !ok {
			return nil, err
		}
		return parts(ctx)
	}
}
