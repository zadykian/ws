package setup_test

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
)

func TestNetworksReportNothingWithoutAPeer(t *testing.T) {
	if got := mactest.ReportOf(t, networksStep(effecttest.NewMachine(), conf.Conf{})); len(got) != 0 {
		t.Errorf("items = %v, want none", got)
	}
}

func TestNetworksReportsRoutesAndResolvers(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/"+daemonName] = effecttest.Result{}
	got := mactest.ReportOf(t, networksStep(m, conf.Conf{PublicKey: "K", TunnelNetworks: "10.1.0.0/24", Resolvers: "a.test=10.0.0.1"}))
	for _, name := range []string{"/etc/ws-tunnel/routes.d/mac-conf", "/etc/resolver", "/etc/resolver/a.test"} {
		if it, ok := got[name]; !ok || it.Found != "missing" {
			t.Errorf("%s = %+v, %v, want it missing", name, it, ok)
		}
	}
}

func TestNetworksReportsAResolverToRemove(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/etc/resolver/gone.test"] = []byte(resolverHead + "nameserver 10.0.0.1\n")
	m.Root["/etc/resolver"], m.Dirs["/etc/resolver"] = effect.Attrs{Mode: 0o755}, 0o755
	m.Root["/etc/resolver/gone.test"] = effect.Attrs{Mode: 0o644}
	it := mactest.ReportOf(t, networksStep(m, conf.Conf{PublicKey: "K"}))["/etc/resolver/gone.test"]
	if it.Kind != step.FileItem || it.Want != "missing" || it.Found != "present" {
		t.Errorf("resolver = %+v, want it to go", it)
	}
}
