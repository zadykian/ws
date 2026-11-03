package jetbrains

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/host"
)

// Backend is one backend's latest build in place, and the other builds of its product removed.
type Backend struct {
	S Site
	B Build
}

// installed reports whether the build's directory holds the marker.
func (b Backend) installed() (bool, error) {
	_, err := b.S.M.Stat(b.B.Dir + "/" + marker)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

// Plan installs the build where it is missing, then removes the builds it replaces. A directory of
// the build without the marker may be one Gateway is unpacking, or a build that a later Gateway
// marks otherwise: a process that runs from it stops the run, rather than lose its files. The run's
// own unpack goes in DIR.part, which no command line names as DIR/.
func (b Backend) Plan(ctx context.Context) ([]host.Change, error) {
	in, err := b.installed()
	if err != nil {
		return nil, err
	}
	var changes []host.Change
	if !in {
		sum, err := b.sum(ctx)
		if err != nil {
			return nil, err
		}
		pids, err := b.S.running(ctx, b.B.Dir)
		if err != nil {
			return nil, err
		}
		if len(pids) > 0 {
			return nil, fmt.Errorf("%s lacks %s, and process %s runs from it: Gateway may be unpacking it. "+
				"Run again once that process ends", b.B.Dir, marker, strings.Join(pids, ", "))
		}
		changes = append(changes, install{s: b.S, b: b.B, sum: sum})
	}
	if !b.S.Prune {
		return changes, nil
	}
	removals, err := others(ctx, b.S, b.B)
	return append(changes, removals...), err
}

// sum reads the SHA-256 that JetBrains publishes for the archive.
func (b Backend) sum(ctx context.Context) (string, error) {
	body, err := b.S.get(ctx, b.B.ChecksumLink)
	if err != nil {
		return "", err
	}
	return ParseSum(body, b.B.ChecksumLink)
}
