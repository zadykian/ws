package wizard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/internal/keychain"
)

// noSum is the sum of no secrets file, in the run's stamp and in the stage's own.
const noSum = "-"

// macSecrets reads the Mac's secrets file, or nil where there is none.
func (r *WsRun) macSecrets() ([]byte, error) {
	if r.Secrets == "" {
		return nil, nil
	}
	data, err := r.Host.M.ReadFile(r.Secrets)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

// localSum returns the SHA-256 of the Mac's secrets file, or noSum where there is none. A run is
// done only where its stamp names the same sum.
func (r *WsRun) localSum() (string, error) {
	data, err := r.macSecrets()
	if err != nil || data == nil {
		return noSum, err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

// stampSum returns the sum that ws's last full run names in its stamp.
func (r *WsRun) stampSum() string {
	if f := strings.Fields(r.Remote.Get("run")); len(f) > 3 {
		return f[3]
	}
	return ""
}

// send puts the Mac's secrets file and the identity on ws, where the remote script's run hands the
// identity to the unit as a credential and removes it. It removes ws's file where the Mac has none.
// Each run of the unit needs the identity again, so send comes before each. The identity goes by
// ssh's stdin, never in an argument.
func (r *WsRun) send(ctx context.Context) error {
	data, err := r.macSecrets()
	if err != nil {
		return err
	}
	if data == nil {
		_, err = r.Remote.Do(ctx, "secrets", "drop")
		return err
	}
	k := keychain.Keychain{M: r.Host.M, User: r.User}
	key, err := k.Key(ctx)
	if errors.Is(err, keychain.ErrNoItem) {
		return fmt.Errorf("%s: %s is there, but the Keychain lacks the item %s: run ws mac secrets init, "+
			"and set the secrets again", runStage, r.Secrets, keychain.Service)
	}
	if err != nil {
		return fmt.Errorf("%s: %w", runStage, err)
	}
	out, err := r.Remote.Do(ctx, "secrets", "stage")
	if err != nil {
		return err
	}
	paths := strings.Fields(string(out))
	if len(paths) != 2 {
		return fmt.Errorf("%s: the remote script named %d paths for the secrets, want 2", runStage, len(paths))
	}
	r.Host.Out.Say("%s: sending the secrets file and the identity to ws", runStage)
	for i, in := range [][]byte{data, []byte(key + "\n")} {
		if _, err := r.Host.SSH.RunInput(ctx, in, "-o", "BatchMode=yes", "ws", "umask 077 && cat >"+paths[i]); err != nil {
			r.drop(ctx)
			return fmt.Errorf("%s: sending %s to ws failed", runStage, [...]string{"the secrets file", "the identity"}[i])
		}
	}
	return nil
}

// drop removes what send put on ws, or its ARGs alone, once more than the remote script's own exit
// does. It is best effort, and it goes on where ctx ends, as a Ctrl-C ends it.
func (r *WsRun) drop(ctx context.Context, only ...string) {
	_, _ = r.Remote.Do(context.WithoutCancel(ctx), append([]string{"secrets", "drop"}, only...)...)
}

// run sends the secrets, then runs ws server apply through the remote script.
func (r *WsRun) run(ctx context.Context) error {
	if err := r.send(ctx); err != nil {
		return err
	}
	if err := r.Remote.Pass(ctx, "run"); err != nil {
		r.drop(ctx, "identity")
		return fail()
	}
	return nil
}
