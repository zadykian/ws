package network

import "github.com/zadykian/ws/engine/core/effect/effecttest"

// scratchDir is the scratch root that the fake machine's mktemp makes.
const scratchDir = "/tmp/x.netplan"

// rigMachine is a machine whose netplan accepts the direct link and finds the port alone.
func rigMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/etc/netplan"] = 0o755
	m.Dirs["/lib/netplan"] = 0o755
	m.Results["mktemp -d --suffix=.netplan"] = effecttest.Result{Out: []byte(scratchDir + "\n")}
	m.Results["cp -a /etc/netplan/. "+scratchDir+"/etc/netplan/"] = effecttest.Result{}
	m.Results["cp -a /lib/netplan/. "+scratchDir+"/lib/netplan/"] = effecttest.Result{}
	m.Results["/usr/libexec/netplan/configure --root-dir "+scratchDir] = effecttest.Result{}
	m.Results["netplan get --root-dir "+scratchDir] = effecttest.Result{Out: []byte("network:\n  ethernets:\n    enp5s0: {}\n")}
	return m
}
