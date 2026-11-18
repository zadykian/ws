package tunnel

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/systemd"
)

// wg0Files are networkd's two files for wg0.
//
//go:embed files/60-wg0.netdev.tmpl files/60-wg0.network.tmpl
var wg0Files embed.FS

// wg0 puts the files in force. networkd creates a missing wg0, and sets up again the links whose
// files are new or changed, and no other. A peer's change updates wg0 in place, but networkd
// replaces all its peers at once, which ends the tunnel's sessions: each peer handshakes again as
// it next sends. Without systemd booted, as in CI's container, it plans nothing.
//
// networkd reads the private key as it creates wg0, and again as a reload finds the .netdev
// changed, but not where the key's file alone changed. So a key other than wg0's takes effect once
// wg0 is deleted and networkd creates it again.
type wg0 struct {
	m effect.Machine
	// watch are the key and the files: a change of any reloads networkd.
	watch []resource.Resource
}

// Plan plans the deletion of wg0 where it holds another key than the file's, or the file is to be
// made. It plans the reload where a watched resource changes or wg0 is missing, as a key that
// networkd couldn't read or a deletion by hand leaves it.
func (w wg0) Plan(ctx context.Context) ([]resource.Change, error) {
	if !systemd.Booted(w.m) {
		return nil, nil
	}
	_, statErr := w.m.Stat("/sys/class/net/wg0")
	exists := statErr == nil
	if statErr != nil && !errors.Is(statErr, fs.ErrNotExist) {
		return nil, statErr
	}
	why, remove, err := w.why(ctx, exists)
	if err != nil || why == "" {
		return nil, err
	}
	var cs []resource.Change
	if remove {
		cs = append(cs, deleteLink{w.m})
	}
	return append(cs, reloadNetworkd{w.m, why}), nil
}

// why says what makes the reload due, and whether wg0 is to be deleted first. It returns an empty
// string where nothing does.
func (w wg0) why(ctx context.Context, exists bool) (string, bool, error) {
	public, hasKey, err := serverPublicKey(w.m)
	if err != nil {
		return "", false, err
	}
	switch {
	case !exists:
		return "wg0 is missing", false, nil
	case !hasKey:
		return "the key is new", true, nil
	}
	out, err := w.m.RunRoot(ctx, "wg", "show", "wg0", "public-key")
	if err != nil {
		return "", false, err
	}
	if strings.TrimSpace(string(out)) != public {
		return "wg0 holds another key than the file's", true, nil
	}
	for _, r := range w.watch {
		cs, err := r.Plan(ctx)
		if err != nil {
			return "", false, err
		}
		if len(cs) > 0 {
			return "a file changed", false, nil
		}
	}
	return "", false, nil
}

// WG0 makes wg0 again where it holds another key than the file's, and waits for it to hold the
// file's. networkd creates a missing wg0 as it reloads. It waits Tries times, Pause apart, by Sleep,
// time.Sleep where nil. The files' own reload is their service's, so WG0 watches no file. Without
// systemd booted, as in CI's container, it plans nothing.
type WG0 struct {
	M     effect.Machine
	Out   effect.Out
	Tries int
	Pause time.Duration
	Sleep func(time.Duration)
}

// Plan plans the deletion of a wg0 of another key, the reload of networkd where wg0 lacks the
// file's key, and the wait for it. A missing key file is an error, as tunnel key makes it first.
func (w WG0) Plan(ctx context.Context) ([]resource.Change, error) {
	if !systemd.Booted(w.M) {
		return nil, nil
	}
	public, hasKey, err := serverPublicKey(w.M)
	if err != nil {
		return nil, err
	}
	if !hasKey {
		return nil, fmt.Errorf("%s is missing: ws server ubuntu tunnel key makes it", keyFile)
	}
	cs, err := wg0{m: w.M}.Plan(ctx)
	if err != nil || len(cs) == 0 {
		return nil, err
	}
	sleep := w.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	r := ready{m: w.M, out: w.Out, tries: w.Tries, pause: w.Pause, sleep: sleep}
	return append(cs, waitForKey{r, public}), nil
}
