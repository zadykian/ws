package tunnel

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// WG0 puts the key in force. networkd reads the private key as it creates wg0, and again as a
// reload finds the .netdev changed, but not where the key's file alone changed. So a key other
// than wg0's takes effect once wg0 is deleted and networkd creates it again, as it reloads. A
// missing wg0, as a key that networkd couldn't read or a deletion by hand leaves it, is created by
// the reload alone. networkd creates wg0 a moment after the reload, so WG0 waits for it: Tries
// times, Pause apart, by Sleep, time.Sleep where nil. The files' own reload is their service's.
type WG0 struct {
	M     effect.Machine
	Out   effect.Out
	Tries int
	Pause time.Duration
	Sleep func(time.Duration)
	// Name and File are the interface and its key's path, wg0 and keyFile where empty.
	Name, File string
}

func (w WG0) name() string {
	if w.Name == "" {
		return "wg0"
	}
	return w.Name
}

func (w WG0) file() string {
	if w.File == "" {
		return keyFile
	}
	return w.File
}

// Plan plans the deletion of a wg0 of another key than the file's, then the reload and the wait
// where wg0 lacks the file's key. A missing key file is an error, as tunnel key makes it first.
func (w WG0) Plan(ctx context.Context) ([]resource.Change, error) {
	public, hasKey, err := publicKeyOf(w.M, w.file())
	if err != nil {
		return nil, err
	}
	if !hasKey {
		return nil, fmt.Errorf("%s is missing: tunnel key makes it", w.file())
	}
	_, statErr := w.M.Stat("/sys/class/net/" + w.name())
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return nil, statErr
	}
	why, remove := w.name()+" is missing", false
	if statErr == nil {
		out, err := w.M.RunRoot(ctx, "wg", "show", w.name(), "public-key")
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(out)) == public {
			return nil, nil
		}
		why, remove = w.name()+" holds another key than the file's", true
	}
	var cs []resource.Change
	if remove {
		cs = append(cs, deleteLink{w.M, w.name()})
	}
	return append(cs, reloadNetworkd{w.M, w.name(), why}, waitForKey{w, public}), nil
}
