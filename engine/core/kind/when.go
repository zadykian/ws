package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// when is a unit's condition: each key a fact of the machine, and its value what the fact must
// give. The unit is on where every key holds. A spec writes it `when = { FACT = "WANT" }`.
type when map[string]string

// fact reports whether the machine gives want. A machine that lacks the fact's source, as macOS
// lacks sysfs, gives none.
type fact func(ctx context.Context, m effect.Machine, want string) (bool, error)

// whenFacts are the facts that a when may name. A new fact is one entry here, its reader below.
var whenFacts = map[string]fact{
	"board_vendor": boardVendor,
	"gpu_vendor":   gpuVendor,
}

// check refuses a key that no fact has, as the spec loads.
func (w when) check() error {
	for _, key := range slices.Sorted(maps.Keys(w)) {
		if _, ok := whenFacts[key]; !ok {
			return fmt.Errorf("when: no fact %q; the facts are %s", key,
				strings.Join(slices.Sorted(maps.Keys(whenFacts)), ", "))
		}
	}
	return nil
}

// gate makes the unit's parts only where every key holds, and says why where one does not. It
// reads the facts as the unit plans, so that a fact can follow an earlier unit. An empty want
// holds on no machine, so that a value left empty turns the unit off.
func (w when) gate(e Env, unit string, parts func(context.Context) ([]resource.Resource, error)) func(context.Context) ([]resource.Resource, error) {
	if len(w) == 0 {
		return parts
	}
	return func(ctx context.Context) ([]resource.Resource, error) {
		for _, key := range slices.Sorted(maps.Keys(w)) {
			want := w[key]
			ok := false
			if want != "" {
				var err error
				if ok, err = whenFacts[key](ctx, e.M, want); err != nil {
					return nil, fmt.Errorf("when %s: %w", key, err)
				}
			}
			if !ok {
				e.Out.Say("%s: off on this machine, as its %s is not %q", unit, key, want)
				return nil, nil
			}
		}
		return parts(ctx)
	}
}

// boardVendorFile is where DMI gives the board's maker.
const boardVendorFile = "/sys/class/dmi/id/board_vendor"

// boardVendor holds where DMI names the board's maker want.
func boardVendor(_ context.Context, m effect.Machine, want string) (bool, error) {
	text, err := m.ReadFile(boardVendorFile)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(text)) == want, nil
}

// pciDevices holds a directory per PCI device, with its class and vendor.
const pciDevices = "/sys/bus/pci/devices"

// gpuVendor holds where a PCI display device, of class 0x03, has the vendor id want, as sysfs
// writes it, such as 0x1234.
func gpuVendor(_ context.Context, m effect.Machine, want string) (bool, error) {
	names, err := m.ReadDir(pciDevices)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, name := range names {
		class, err := m.ReadFile(pciDevices + "/" + name + "/class")
		if err != nil {
			return false, err
		}
		vendor, err := m.ReadFile(pciDevices + "/" + name + "/vendor")
		if err != nil {
			return false, err
		}
		if strings.HasPrefix(string(class), "0x03") && strings.TrimSpace(string(vendor)) == want {
			return true, nil
		}
	}
	return false, nil
}
