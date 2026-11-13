package cmd

import (
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// wizardProcess returns a root process on a machine that holds vars as host_vars, booted or not,
// whose remote script reports status as host_vars' ws_user reads it.
func wizardProcess(vars string, booted bool, status string) (cli.Env, *effecttest.Machine) {
	p, m := valuesProcess(vars)
	p.GOOS = "linux"
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	v, _ := values.Load(m, serverVars)
	m.Results["env SUDO_USER="+v.User+" sh -s -- status"] = effecttest.Result{Out: []byte(status)}
	return p, m
}

// valuesProcess returns a root process on a machine that holds file as the values.
func valuesProcess(file string) (cli.Env, *effecttest.Machine) {
	p := fakeProcess()
	m := effecttest.NewMachine()
	p.Machine, p.Euid = m, 0
	m.Dirs["/etc/ws"] = 0o755
	m.Files[serverVars] = []byte(file)
	return p, m
}
