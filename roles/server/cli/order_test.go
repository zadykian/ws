package cli

import (
	"os"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/roles/server/values"
)

// before fails the test where a unit of first does not run ahead of every unit of then, in the
// units' order for root and for the account.
func before(t *testing.T, first string, then ...string) {
	root, account := rootAndAccount(t)
	for _, names := range [][]string{specLines(t, root), specLines(t, account)} {
		i := slices.Index(names, first)
		for _, n := range then {
			if j := slices.Index(names, n); i < 0 || j < 0 || i > j {
				t.Errorf("%s at %d, %s at %d; want %s first", first, i, n, j, first)
			}
		}
	}
}

// The collector's units run after signoz's units, so that SigNoz's retention is set before the
// collector sends. The endpoint they read is in /etc/ws/env, which mise writes before any unit.
func TestOtelcolRunsAfterSignoz(t *testing.T) {
	before(t, "state/signoz-password", "deb/otelcol")
	before(t, "state/signoz-sync", "deb/otelcol")
	before(t, "deb/otelcol", "service/otelcol")
}

// A read takes a unit as pending where a unit it waits on, by its chain of after, has changes. The
// collector's service starts after SigNoz's sync, so its chain holds it. signoz's commands run ws,
// so theirs holds the frame's unit.
func TestOtelcolWaitsOnSignoz(t *testing.T) {
	root, account := rootAndAccount(t)
	for _, v := range []values.Values{root, account} {
		e := kind.Env{M: effecttest.NewMachine(), U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v, Self: selfPath}
		units, err := kind.Load(e, os.DirFS(rolesDir))
		if err != nil {
			t.Fatal(err)
		}
		for unit, wants := range map[string][]string{
			"service/otelcol":       {"deb/otelcol", "state/signoz-sync", "state/ws-binary"},
			"state/signoz-password": {"state/ws-binary"},
		} {
			chain, err := kind.Closure(units, []string{unit})
			if err != nil {
				t.Fatal(err)
			}
			for _, w := range wants {
				if !slices.Contains(kind.Names(chain), w) {
					t.Errorf("%s for %s: its chain %q lacks %s", unit, v.User, kind.Names(chain), w)
				}
			}
		}
	}
}

// SigNoz's project runs on the system's daemon, which docker's units set up.
func TestSignozRunsAfterDocker(t *testing.T) {
	before(t, "service/docker", "compose/signoz")
}

// git's credential program of each token is /usr/local/bin/ws, which the frame's unit keeps. A
// first run that clones by a token would otherwise call a missing program and stop there.
func TestGitTokensRunAfterWs(t *testing.T) {
	before(t, "state/ws-binary", "state/git-tokens-group", "state/git-tokens", "state/git-clones")
}

// signoz's commands run the ws that the frame's unit keeps.
func TestSignozCommandsRunAfterWs(t *testing.T) {
	before(t, "state/ws-binary", "state/signoz-password", "state/signoz-sync")
	before(t, "state/signoz-password", "compose/signoz")
	before(t, "compose/signoz", "state/signoz-sync")
}
