package otelcol_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/ubuntu/values"
)

// machine is a machine with the collector's directories and /etc/ws/env set to env, and
// the environment that loads the role's units over it.
func machine(t *testing.T, env string) (*effecttest.Machine, kind.Env) {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	return machineOf(t, env, v)
}

// machineOf is machine for a side's own values, such as the host's.
func machineOf(t *testing.T, env string, data any) (*effecttest.Machine, kind.Env) {
	m := effecttest.NewMachine()
	for _, d := range []string{"/etc", "/etc/ws", "/etc/systemd", "/etc/systemd/system", "/etc/otelcol-contrib"} {
		m.Dirs[d] = 0o755
	}
	if env != "" {
		m.Files[envPath] = []byte(env)
	}
	for _, p := range []string{envFile, config} {
		m.Results[checkLine(t, p)] = effecttest.Result{}
	}
	return m, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: data}
}
