package wireguard

import (
	"context"
	"embed"
	"errors"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
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
	watch []host.Resource
}

// Plan plans the deletion of wg0 where it holds another key than the file's, or the file is to be
// made. It plans the reload where a watched resource changes or wg0 is missing, as a key that
// networkd couldn't read or a deletion by hand leaves it.
func (w wg0) Plan(ctx context.Context) ([]host.Change, error) {
	if !host.Booted(w.m) {
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
	var cs []host.Change
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
