package wizard

import (
	"context"
	"strings"
)

// noSum is the sum of no secrets file, in the run's stamp and in the remote script's status.
const noSum = "-"

// sameSecrets tells whether ws's last full run read ws's secrets file as it is now: the run's stamp
// names the sum that the status reads, which is noSum in both where the run had no file to read.
// So the wizard reads one sum, whichever machine it runs on.
func (r *WsRun) sameSecrets() bool {
	f := strings.Fields(r.Remote.Get("run"))
	return len(f) > 3 && f[3] == r.Remote.Get("secrets")
}

// run runs ws server apply through the remote script.
func (r *WsRun) run(ctx context.Context) error {
	if err := r.Remote.Follow(ctx, onlyArgs(r.Only)...); err != nil {
		// A screen ends by the cut's own words, which leave the unit unsaid, and a plain run by fail's.
		if ctx.Err() != nil {
			r.Host.Out.ByHand("%s: the unit goes on on ws; run the wizard again, which follows it", runStage)
		}
		return fail()
	}
	return nil
}

// onlyArgs returns --only for each entry. A step's name may hold a blank: the Shell keeps it whole.
func onlyArgs(only []string) []string {
	args := make([]string, len(only))
	for i, entry := range only {
		args[i] = "--only=" + entry
	}
	return args
}
