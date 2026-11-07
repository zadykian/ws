package wizard

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/internal/keychain"
)

// noSum is the sum of no secrets file, in the run's stamp and in the remote script's status.
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

// sameSecrets tells whether ws's last full run read ws's secrets file as it is now: the run's stamp
// names the sum that the status reads, which is noSum in both where the run had no file to read.
// So the wizard reads one sum, whichever machine it runs on.
func (r *WsRun) sameSecrets() bool {
	f := strings.Fields(r.Remote.Get("run"))
	return len(f) > 3 && f[3] == r.Remote.Get("secrets")
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
		if err := r.Remote.Put(ctx, paths[i], in); err != nil {
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
	if err := r.Remote.Follow(ctx, onlyArgs(r.Only)...); err != nil {
		r.drop(ctx, "identity")
		// A screen ends by the cut's own words, which leave the unit unsaid, and a plain run by fail's.
		if ctx.Err() != nil {
			r.Host.Out.ByHand("%s: the unit goes on on ws; run the wizard again, which follows it", runStage)
		}
		return fail()
	}
	return nil
}

// onlyArgs returns --only for each entry, quoted for ws's shell, as a step's name may hold a blank.
func onlyArgs(only []string) []string {
	args := make([]string, len(only))
	for i, entry := range only {
		args[i] = "'--only=" + strings.ReplaceAll(entry, "'", `'\''`) + "'"
	}
	return args
}
