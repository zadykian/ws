package wizard

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/internal/sshcfg"
)

// other returns the key of Secretive's agent other than skip's .pub, or none, and fails where it
// lists several. An agent that does not answer lists none, as while Secretive starts.
func (k *MacKey) other(ctx context.Context, skip string) (sshcfg.AgentKey, error) {
	keys, _, _ := setup.SecretiveKeys(ctx, k.M, k.Home)
	var got sshcfg.AgentKey
	for _, key := range keys {
		if key.Name == path.Base(skip) {
			continue
		}
		if got.Name != "" {
			return got, fmt.Errorf("%s: Secretive holds several keys, %s and %s: name ws's in mac.conf's WS_IDENTITY, "+
				"and git's in WS_GIT_SIGNING_KEY", macKey, sshcfg.Expand(got.Pub(), k.Home), sshcfg.Expand(key.Pub(), k.Home))
		}
		got = key
	}
	return got, nil
}

// pick writes ws's copy of the .pub of the key of Secretive's agent other than skip's, and returns
// its path. Where the agent lists none, it says to make what in Secretive's app, adds extra, and
// waits up to 15 minutes for one. It fails on a key other than P-256, as an ML-DSA one, which ws's
// OpenSSH lacks.
func (k *MacKey) pick(ctx context.Context, skip, what, extra string) (string, error) {
	got, err := k.other(ctx, skip)
	if err == nil && got.Name == "" {
		if extra != "" {
			extra = ". " + extra
		}
		k.Out.Say("%s: in Secretive, make %s; the wizard waits up to 15 minutes for it%s", macKey, what, extra)
		for left := 450; got.Name == "" && err == nil; got, err = k.other(ctx, skip) {
			// A signal ends the wait at once, as a trap did in sh.
			if err := ctx.Err(); err != nil {
				return "", err
			}
			if left--; left < 0 {
				return "", fmt.Errorf("%s: Secretive made no key in 15 minutes: make %s, then run the wizard again%s",
					macKey, what, extra)
			}
			k.Sleep(2 * time.Second)
		}
	}
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(firstFields(got.Line), "ecdsa-sha2-nistp256 ") {
		return "", fmt.Errorf("%s: %s is not the P-256 key asked for: remove it in Secretive, make %s, then run the "+
			"wizard again", macKey, got.Name, what)
	}
	pub := sshcfg.Expand(got.Pub(), k.Home)
	return pub, setup.Apply(ctx, k.M, k.Out, setup.KeyCopy{Path: pub, Text: got.Line + "\n"})
}
