package setup_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
)

const (
	tunnelKey  = "/etc/ws-tunnel/private.key"
	rootWs     = "/usr/local/libexec/ws-tunnel/ws"
	daemonName = "com.github.zadykian.ws-tunnel"
)

func TestCableReportsItsAddress(t *testing.T) {
	const list = "header\nDock\n"
	m := effecttest.NewMachine()
	it := mactest.ReportOf(t, cableStep(m, &effecttest.Out{}, list, cableSet))["Dock"]
	if it.Kind != step.ServiceItem || it.Found != it.Want {
		t.Errorf("set = %+v, want its address found", it)
	}
	it = mactest.ReportOf(t, cableStep(m, &effecttest.Out{}, list, "DHCP Configuration\nIP address: 10.0.0.9\n"))["Dock"]
	if it.Found == it.Want || !strings.Contains(it.Found, "10.0.0.9") {
		t.Errorf("dhcp = %+v, want the address it has", it)
	}
	it = mactest.ReportOf(t, cableStep(m, &effecttest.Out{}, "header\n*Dock\n", cableSet))["Dock"]
	if it.Found == it.Want || !strings.Contains(it.Conflict, "disabled") {
		t.Errorf("disabled = %+v, want a conflict: no run turns it on", it)
	}
	s := cableStep(m, &effecttest.Out{}, list, cableSet)
	s.Conf = conf.Conf{}
	if got := mactest.ReportOf(t, s); len(got) != 0 {
		t.Errorf("items = %v, want none without a service", got)
	}
}

func TestTunnelReportsAMacThatIsDone(t *testing.T) {
	m := daemonMachine()
	got := mactest.ReportOf(t, daemonStep(m))
	for _, name := range []string{
		"/h/.local/bin/ws", rootWs, "/etc/ws-tunnel", "/etc/ws-tunnel/routes.d", tunnelKey,
		"/h/.config/ws/tunnel.pub", "/etc/ws-tunnel/tunnel.conf", daemonName,
		"/Library/LaunchDaemons/" + daemonName + ".plist",
	} {
		if it, ok := got[name]; !ok || it.Found != it.Want || it.Conflict != "" || it.Diff != "" {
			t.Errorf("%s = %+v, %v, want it right", name, it, ok)
		}
	}
	if got[tunnelKey].Kind != step.FileItem || got[daemonName].Kind != step.ServiceItem {
		t.Errorf("key = %+v, daemon = %+v", got[tunnelKey], got[daemonName])
	}
}

func TestTunnelReportNeverReadsTheKey(t *testing.T) {
	m := daemonMachine()
	mactest.ReportOf(t, daemonStep(m))
	if slices.Contains(m.Calls, "read root "+tunnelKey) || slices.Contains(m.Calls, "read "+tunnelKey) {
		t.Errorf("calls = %q, which read the private key", m.Calls)
	}
	asked := 0
	for _, c := range m.Calls {
		if strings.Contains(c, "client macos tunnel key") {
			asked++
		}
	}
	if asked != 1 {
		t.Errorf("root's ws was asked for the public key %d times, want once, by Plan", asked)
	}
}

func TestTunnelReportShowsNoDiffOfAProgram(t *testing.T) {
	m := daemonMachine()
	m.Files[rootWs] = []byte("old")
	m.Files["/h/.local/bin/ws"] = []byte("old")
	got := mactest.ReportOf(t, daemonStep(m))
	for _, name := range []string{rootWs, "/h/.local/bin/ws"} {
		if it := got[name]; it.Found == it.Want || it.Diff != "" {
			t.Errorf("%s = %+v, want it to differ, with no diff", name, it)
		}
	}
}

func TestTunnelReportsADaemonLaunchdLacks(t *testing.T) {
	m := daemonMachine()
	m.Results["launchctl print system/"+daemonName] = effecttest.Result{Err: errors.New("no")}
	if it := mactest.ReportOf(t, daemonStep(m))[daemonName]; it.Found != "not loaded" {
		t.Errorf("daemon = %+v", it)
	}
}

func TestTunnelWithoutAPeerReportsNoDaemon(t *testing.T) {
	got := mactest.ReportOf(t, tunnelStep(effecttest.NewMachine()))
	if _, ok := got[daemonName]; ok {
		t.Errorf("items = %v, want no daemon before mac.conf names a peer", got)
	}
	if it := got[tunnelKey]; it.Kind != step.FileItem || it.Found != "missing" {
		t.Errorf("key = %+v", it)
	}
}

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
