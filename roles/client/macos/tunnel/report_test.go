package tunnel_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
)

const (
	tunnelKey  = "/etc/ws-tunnel/private.key"
	rootWs     = "/usr/local/libexec/ws-tunnel/ws"
	daemonName = "com.github.zadykian.ws-tunnel.ws"
)

func TestTunnelReportsAMacThatIsDone(t *testing.T) {
	m := daemonMachine()
	got := mactest.ReportOf(t, daemonStep(m))
	for _, name := range []string{
		"/h/.local/bin/ws", rootWs, "/etc/ws-tunnel", "/etc/ws-tunnel/routes.d", "/etc/ws-tunnel/ws", tunnelKey,
		"/h/.config/ws/tunnel.pub", "/etc/ws-tunnel/ws/tunnel.conf", daemonName,
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
