package kind

import (
	"context"
	"errors"
	"io/fs"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// modprobeName is a module's name as modprobe takes it: no path, and no leading dash that would
// read as an option.
var modprobeName = regexp.MustCompile(`^[a-z0-9_][a-z0-9_-]*$`)

// dkmsModules unloads the modules that hold a dkms unit's device, and loads its own. With Secure
// Boot on, the kernel refuses a module that DKMS built until DKMS's key is enrolled, by hand at the
// console. The run says so and goes on. A refusal is marked in /run, which the reboot that enrolls
// the key clears, so that the runs between plan no load.
type dkmsModules struct {
	m      effect.Machine
	out    effect.Out
	load   string
	unload []string
	hint   string
}

// refusedMark is the mark of the kernel's refusal of a module.
func refusedMark(module string) string { return "/run/ws-refused-" + module }

// loaded reports whether the kernel holds the module, which /sys/module names with _ for -.
func (d dkmsModules) loaded(name string) (bool, error) {
	_, err := d.m.Stat("/sys/module/" + strings.ReplaceAll(name, "-", "_"))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func (d dkmsModules) Plan(context.Context) ([]resource.Change, error) {
	var changes []resource.Change
	for _, name := range d.unload {
		if held, err := d.loaded(name); err != nil {
			return nil, err
		} else if held {
			changes = append(changes, asRoot{d.m, "unload the module " + name, []string{"modprobe", "-r", name}})
		}
	}
	if ours, err := d.loaded(d.load); err != nil || ours {
		return changes, err
	}
	if _, err := d.m.Stat(refusedMark(d.load)); err == nil {
		return changes, nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	return append(changes, loadModule{d}), nil
}

// loadModule loads the module, and notes the kernel's refusal under Secure Boot without failing.
type loadModule struct{ d dkmsModules }

func (c loadModule) String() string { return "load the module " + c.d.load }

func (c loadModule) Apply(ctx context.Context) error {
	_, err := c.d.m.RunRoot(ctx, "modprobe", c.d.load)
	if err == nil || !strings.Contains(err.Error(), "Key was rejected") {
		return err
	}
	c.d.out.Say("%s", strings.TrimSpace("the kernel refused "+c.d.load+
		", as Secure Boot is on and DKMS's key is not enrolled. "+c.d.hint))
	_, err = c.d.m.RunRoot(ctx, "touch", "--", refusedMark(c.d.load))
	return err
}
