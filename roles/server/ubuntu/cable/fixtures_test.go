package cable

import (
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const (
	// scratchDir is the scratch root that the fake machine's mktemp makes.
	scratchDir = "/tmp/x.netplan"
	linkJSON   = `[{"ifname":"enp5s0","address":"00:1a:2b:3c:4d:5e","addr_info":[]}]`
	ipPrefix   = "ip -j "
)

// rigMachine is a machine whose netplan accepts the cable and finds the port alone.
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

// portMachine is rigMachine with the port enp5s0, no default route and networkd running.
func portMachine() (*effecttest.Machine, *effecttest.Units) {
	m := rigMachine()
	m.Dirs["/sys/class/net/enp5s0"] = 0o755
	m.Results[ipPrefix+"-4 route show default table all"] = effecttest.Result{Out: []byte("[]")}
	m.Results[ipPrefix+"-6 route show default table all"] = effecttest.Result{Out: []byte("[ ]")}
	m.Results[ipPrefix+"address show dev enp5s0"] = effecttest.Result{Out: []byte(linkJSON)}
	m.Results["udevadm info --query=property --property=ID_NET_DRIVER --value /sys/class/net/enp5s0"] = effecttest.Result{Out: []byte("r8169\n")}
	return m, &effecttest.Units{States: map[string]effect.UnitState{"systemd-networkd.service": {Active: true}}}
}
