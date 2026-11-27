// Package timezone keeps the machine's time zone, as tzdata reads it.
package timezone

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// zoneinfo holds a file per zone, which /etc/localtime links to.
const zoneinfo = "/usr/share/zoneinfo/"

// Zone is the time zone Name: the link /etc/localtime, made without timedatectl, which needs a
// running systemd, and /etc/timezone where the machine has it. Hint ends a refusal of the name: how
// to give another.
type Zone struct {
	M    effect.Machine
	Name string
	Hint string
}

// Check refuses a name that is no path below zoneinfo, so that a spec can refuse it as it loads.
func Check(name string) error {
	if name == "" || path.IsAbs(name) || path.Clean(name) != name || name == ".." || strings.HasPrefix(name, "../") {
		return fmt.Errorf("the time zone %q is no name below %s, such as Etc/UTC", name, zoneinfo)
	}
	return nil
}

// Plan plans the link, then /etc/timezone.
func (z Zone) Plan(ctx context.Context) ([]resource.Change, error) { return z.parts().Plan(ctx) }

func (z Zone) parts() resource.Joined { return resource.Joined{zoneLink(z), legacyZone(z)} }

// zoneLink is /etc/localtime, a link to the zone's file.
type zoneLink Zone

// Plan refuses a zone that is no file under zoneinfo.
func (z zoneLink) Plan(ctx context.Context) ([]resource.Change, error) {
	if err := Check(z.Name); err != nil {
		return nil, fmt.Errorf("%w. %s", err, z.Hint)
	}
	target := zoneinfo + z.Name
	if info, err := z.M.Stat(target); errors.Is(err, fs.ErrNotExist) || (err == nil && info.Mode.IsDir()) {
		return nil, fmt.Errorf("the time zone %q: %s is no zone file. %s", z.Name, target, z.Hint)
	} else if err != nil {
		return nil, err
	}
	return file.Link{M: z.M, Path: "/etc/localtime", Target: target}.Plan(ctx)
}

// legacyZone is /etc/timezone. tzdata no longer uses it, but a machine installed with it may keep
// it, and some readers, such as older JVMs and Python's tzlocal, check it before the link.
type legacyZone Zone

// Plan plans the file where it exists, and none where it does not.
func (z legacyZone) Plan(ctx context.Context) ([]resource.Change, error) {
	if _, err := z.M.Stat("/etc/timezone"); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return file.File{M: z.M, Path: "/etc/timezone", Content: []byte(z.Name + "\n"), Mode: 0o644}.Plan(ctx)
}
