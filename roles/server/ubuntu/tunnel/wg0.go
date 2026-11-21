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
}

// Plan plans the deletion of a wg0 of another key than the file's, then the reload and the wait
// where wg0 lacks the file's key. A missing key file is an error, as tunnel key makes it first.
func (w WG0) Plan(ctx context.Context) ([]resource.Change, error) {
	public, hasKey, err := serverPublicKey(w.M)
	if err != nil {
		return nil, err
	}
	if !hasKey {
		return nil, fmt.Errorf("%s is missing: ws server tunnel key makes it", keyFile)
	}
	_, statErr := w.M.Stat("/sys/class/net/wg0")
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return nil, statErr
	}
	why, remove := "wg0 is missing", false
	if statErr == nil {
		out, err := w.M.RunRoot(ctx, "wg", "show", "wg0", "public-key")
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(out)) == public {
			return nil, nil
		}
		why, remove = "wg0 holds another key than the file's", true
	}
	var cs []resource.Change
	if remove {
		cs = append(cs, deleteLink{w.M})
	}
	return append(cs, reloadNetworkd{w.M, why}, waitForKey{w, public}), nil
}
