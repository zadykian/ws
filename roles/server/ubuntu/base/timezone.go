package base

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

// timezoneParts links /etc/localtime to the zone, without timedatectl, which needs a running
// systemd, and keeps /etc/timezone in step where the machine has it.
func timezoneParts(m effect.Machine, zone string) []resource.Resource {
	return []resource.Resource{
		zoneLink{M: m, Zone: zone},
		legacyZone{M: m, Zone: zone},
	}
}

// zoneLink is /etc/localtime, a link to the zone's file.
type zoneLink struct {
	M    effect.Machine
	Zone string
}

// Plan refuses a zone that is no file under /usr/share/zoneinfo.
func (z zoneLink) Plan(ctx context.Context) ([]resource.Change, error) {
	if z.Zone == "" || path.IsAbs(z.Zone) || path.Clean(z.Zone) != z.Zone || strings.HasPrefix(z.Zone, "../") {
		return nil, fmt.Errorf("base_timezone is %q: it takes a name under /usr/share/zoneinfo, such as Etc/UTC", z.Zone)
	}
	target := "/usr/share/zoneinfo/" + z.Zone
	if info, err := z.M.Stat(target); errors.Is(err, fs.ErrNotExist) || (err == nil && info.Mode.IsDir()) {
		return nil, fmt.Errorf("base_timezone is %q: %s is no zone file", z.Zone, target)
	} else if err != nil {
		return nil, err
	}
	return file.Link{M: z.M, Path: "/etc/localtime", Target: target}.Plan(ctx)
}

// legacyZone is /etc/timezone. tzdata no longer uses it, but a machine installed with it may keep
// it, and some readers, such as older JVMs and Python's tzlocal, check it before the link.
type legacyZone struct {
	M    effect.Machine
	Zone string
}

// Plan plans the file where it exists, and none where it does not.
func (z legacyZone) Plan(ctx context.Context) ([]resource.Change, error) {
	if _, err := z.M.Stat("/etc/timezone"); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return file.File{M: z.M, Path: "/etc/timezone", Content: []byte(z.Zone + "\n"), Mode: 0o644}.Plan(ctx)
}
