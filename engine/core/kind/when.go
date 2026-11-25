package kind

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
)

// when is a unit's condition: each key a fact of the machine, and its value what the fact must
// give. The unit is on where every key holds. A spec writes it `when = { FACT = "WANT" }`, or
// `"!WANT"` where the fact must not give WANT.
type when map[string]string

// not marks a want that its fact must not give.
const not = "!"

// want splits a key's value into what the fact gives and whether the unit wants it not given.
func (w when) want(key string) (string, bool) {
	want, negated := strings.CutPrefix(w[key], not)
	return want, negated
}

// fact is one fact that a when may name.
type fact struct {
	// holds reports whether the machine gives want. A machine that lacks the fact's source, as
	// macOS lacks sysfs, gives none.
	holds func(ctx context.Context, e Env, want string) (bool, error)
	// form is the form of a want that the machine can give, or nil for any. A want of another
	// form would turn its unit off on every machine, so the load refuses it.
	form *regexp.Regexp
	// always marks a fact that every machine gives for any want. Its "!WANT" would turn the unit
	// off on every machine, so the load refuses it.
	always bool
}

// whenFacts are the facts that a when may name. A new fact is one entry here, its reader below.
var whenFacts = map[string]fact{
	"board_vendor": {holds: boardVendor},
	// A container shares its host's kernel, so a unit that sets the kernel's state wants "!yes".
	"container": {holds: containerFact, form: regexp.MustCompile(`^yes$`)},
	// sysfs writes a vendor as 0x and four hex digits in lower case.
	"gpu_vendor": {holds: gpuVendor, form: regexp.MustCompile(`^0x[0-9a-f]{4}$`)},
	// The kernel takes an interface's name of 1 to 15 bytes, with no '/', ':' or white space, and
	// neither "." nor "..".
	"interface": {holds: netInterface, form: regexp.MustCompile(`^(?:[^/:\s.][^/:\s]{0,14}|\.[^/:\s.][^/:\s]{0,13}|\.\.[^/:\s]{1,13})$`)},
	// A value of the side's: the unit runs where the value is set, as an empty want is off.
	"set": {holds: anyWant, always: true},
	// The side that ws unit serves, which Env.Side names. Load leaves a unit of another side out.
	"side": {holds: sideFact, form: regexp.MustCompile(`^[a-z]+$`)},
}

// onSide reports whether a unit with this when is one of side: it names no side, or its side fact
// holds for side. An empty want holds on no side.
func (w when) onSide(side string) bool {
	if _, named := w["side"]; !named {
		return true
	}
	want, negated := w.want("side")
	return want != "" && (want == side) != negated
}

// check refuses a key that no fact has, and a want of a form that its fact never gives, as the
// spec loads. An empty want is of any form: it turns the unit off, negated or not.
func (w when) check() error {
	for _, key := range slices.Sorted(maps.Keys(w)) {
		f, ok := whenFacts[key]
		if !ok {
			return fmt.Errorf("when: no fact %q; the facts are %s", key,
				strings.Join(slices.Sorted(maps.Keys(whenFacts)), ", "))
		}
		want, negated := w.want(key)
		if want != "" && f.form != nil && !f.form.MatchString(want) {
			return fmt.Errorf("when: %s %q is not of the form %s", key, w[key], f.form)
		}
		if negated && f.always {
			return fmt.Errorf("when: %s %q holds on no machine, as %s holds for any want", key, w[key], key)
		}
	}
	return nil
}

// gate makes the unit's parts only where every key holds, and says why where one does not. It
// reads the facts as the unit plans, so that a fact can follow an earlier unit. An empty want
// holds on no machine, so that a value left empty turns the unit off. Its line names the empty
// want, and comes before any fact is read, as no fact of the machine is the cause. It says its line
// once, as a read makes the parts twice: to plan, and to report.
func (w when) gate(e Env, unit string, parts func(context.Context) ([]resource.Resource, error)) func(context.Context) ([]resource.Resource, error) {
	if len(w) == 0 {
		return parts
	}
	said := false
	off := func(format string, args ...any) {
		if !said {
			e.Out.Say("%s: off on this machine, as "+format, append([]any{unit}, args...)...)
			said = true
		}
	}
	return func(ctx context.Context) ([]resource.Resource, error) {
		keys := slices.Sorted(maps.Keys(w))
		for _, key := range keys {
			if want, _ := w.want(key); want == "" {
				off("its when gives no %s", key)
				return nil, nil
			}
		}
		for _, key := range keys {
			want, negated := w.want(key)
			gives, err := whenFacts[key].holds(ctx, e, want)
			if err != nil {
				return nil, fmt.Errorf("when %s: %w", key, err)
			}
			if gives == negated {
				if negated {
					off("its %s is %q", key, want)
				} else {
					off("its %s is not %q", key, want)
				}
				return nil, nil
			}
		}
		return parts(ctx)
	}
}

// boardVendorFile is where DMI gives the board's maker.
const boardVendorFile = "/sys/class/dmi/id/board_vendor"

// boardVendor holds where DMI names the board's maker want.
func boardVendor(_ context.Context, e Env, want string) (bool, error) {
	text, err := e.M.ReadFile(boardVendorFile)
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
func gpuVendor(_ context.Context, e Env, want string) (bool, error) {
	names, err := e.M.ReadDir(pciDevices)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	for _, name := range names {
		class, err := e.M.ReadFile(pciDevices + "/" + name + "/class")
		if err != nil {
			return false, err
		}
		vendor, err := e.M.ReadFile(pciDevices + "/" + name + "/vendor")
		if err != nil {
			return false, err
		}
		if strings.HasPrefix(string(class), "0x03") && strings.TrimSpace(string(vendor)) == want {
			return true, nil
		}
	}
	return false, nil
}

// netDevices holds an entry per network interface of the machine's namespace.
const netDevices = "/sys/class/net"

// netInterface holds where the machine has the network interface want, of any state.
func netInterface(_ context.Context, e Env, want string) (bool, error) {
	_, err := e.M.Stat(netDevices + "/" + want)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// containerFact holds where the machine is a container, by the marks that InContainer reads.
func containerFact(_ context.Context, e Env, _ string) (bool, error) {
	return InContainer(e.M), nil
}

// anyWant holds for any want: the fact set's, which only an empty want turns off.
func anyWant(context.Context, Env, string) (bool, error) { return true, nil }

// sideFact holds where the side that ws unit serves is want.
func sideFact(_ context.Context, e Env, want string) (bool, error) { return e.Side == want, nil }
