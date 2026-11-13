package wizard

import (
	"cmp"
	"context"
)

// done tells whether ws's last full run, by this version, exited 0 and no run's unit runs there. On
// a cloud VM, that run must have begun with host_vars turning the outbound table off, which
// host_vars, the same since, still does.
func (s wsStatus) done(version string, c Want) bool {
	return s.idle(version) && s.named(c) && (c.Target != "cloud" || s.Run.Off)
}

// named tells whether ws's values give ws_user the user that ssh logs in as. It compares with the
// server's default, so that a default that moves leaves a WS_USER of the old one named in
// host_vars.
func (s wsStatus) named(c Want) bool { return cmp.Or(s.User, defaultUser()) == c.User }

// idle tells whether ws's last full run, made by a ws of this version, exited 0. The run must have
// read ws's secrets file and host_vars as they are now, and no run's unit may run there now.
func (s wsStatus) idle(version string) bool {
	return s.Active == "" && s.Run.Exit == "0" && s.Run.Version == version && s.sameSecrets() && s.sameVars()
}

// sameSecrets tells whether ws's last full run read ws's secrets file as it stands now. The run's
// stamp names the sum that the status reads, which is empty in both where the run had no file to
// read. So the wizard reads one sum, whichever machine it runs on.
func (s wsStatus) sameSecrets() bool { return s.Run.Exit != "" && s.Run.Secrets == s.Secrets }

// sameVars tells whether ws's last full run read ws's host_vars as it stands now, by the sum that
// the stamp names after the secrets file's. So a change to host_vars, by either wizard, Configure
// or by hand, makes the next run due.
func (s wsStatus) sameVars() bool { return s.Run.Exit != "" && s.Run.Vars == s.Vars }

// run runs ws server ubuntu apply through the remote script.
func (r *WsRun) run(ctx context.Context) error {
	if err := r.Ws.run(ctx, onlyArgs(r.Only)); err != nil {
		// A screen ends by the cut's own words, which leave the unit unsaid, and a plain run by fail's.
		if ctx.Err() != nil {
			r.Out.ByHand("%s: the unit goes on on ws; run the wizard again, which follows it", runStage)
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
