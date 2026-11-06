package setup_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/step"
)

const (
	zedConfig  = "/h/.config/zed/"
	tunnelKey  = "/etc/ws-tunnel/private.key"
	rootWs     = "/usr/local/libexec/ws-tunnel/ws"
	daemonName = "com.github.zadykian.ws-tunnel"
)

func TestACaskThatHomebrewHoldsStandsAgainstWs(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	s := zedStep(m, out)
	delete(m.Dirs, "/h/Applications/Zed.app")
	m.Results["brew list --cask zed"] = effecttest.Result{}
	items, err := s.Report(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if it := items[0]; it.Kind != step.PackageItem || it.Name != "zed" || it.Found != "missing" ||
		!strings.Contains(it.Conflict, "brew reinstall --cask zed") {
		t.Errorf("cask = %+v, want it missing and against ws", it)
	}
	if len(out.Lines) != 0 {
		t.Errorf("out = %q: a report says nothing", out.Lines)
	}
}

func TestACaskAbsentFromHomebrewIsPendingAlone(t *testing.T) {
	m := effecttest.NewMachine()
	s := zedStep(m, &effecttest.Out{})
	delete(m.Dirs, "/h/Applications/Zed.app")
	m.Results["brew list --cask zed"] = effecttest.Result{Err: errors.New("no cask")}
	items, err := s.Report(context.Background())
	if err != nil || items[0].Found != "missing" || items[0].Conflict != "" {
		t.Errorf("items = %+v, %v, want the cask missing, with no conflict", items, err)
	}
}

func TestZedReportKeepsAForeignFileOut(t *testing.T) {
	m := effecttest.NewMachine()
	s := zedStep(m, &effecttest.Out{})
	m.Files[zedConfig+"keymap.json"] = []byte("[ // mine\n]\n")
	m.Files[zedConfig+"settings.json"] = []byte("{ // mine\n}\n")
	got := reportOf(t, s)
	it := got[zedConfig+"keymap.json"]
	if !strings.Contains(it.Conflict, "not ws's") || it.Diff != "" || strings.Contains(fmt.Sprintf("%+v", it), "mine") {
		t.Errorf("keymap = %+v, want a conflict that shows none of the file", it)
	}
	if it := got[zedConfig+"settings.json"]; it.Want != "present" || it.Found != "present" || it.Conflict != "" {
		t.Errorf("settings = %+v, want it found", it)
	}
	if slices.Contains(m.Calls, "read "+zedConfig+"settings.json") {
		t.Errorf("calls = %q, which read Zed's own file", m.Calls)
	}
}

func TestGhosttyReportsAFreshMac(t *testing.T) {
	got := reportOf(t, ghosttyStep(effecttest.NewMachine(), &effecttest.Out{}))
	if it := got["ghostty"]; it.Kind != step.PackageItem || it.Found != "installed" {
		t.Errorf("cask = %+v", it)
	}
	if it := got["font-jetbrains-mono-nerd-font"]; it.Found != "installed" {
		t.Errorf("font = %+v", it)
	}
	for _, name := range []string{ghosttyDir, ghosttyDir + "/ws.ghostty", ghosttyDir + "/config.ghostty"} {
		if it := got[name]; it.Found != "missing" || it.Conflict != "" {
			t.Errorf("%s = %+v, want it missing", name, it)
		}
	}
}

func TestGhosttyReportsAFileChangedByHand(t *testing.T) {
	m := effecttest.NewMachine()
	s := ghosttyStep(m, &effecttest.Out{})
	m.Files[ghosttyDir+"/ws.ghostty"] = []byte(ghosttyHead + "font-size = 20\n")
	it := reportOf(t, s)[ghosttyDir+"/ws.ghostty"]
	if !strings.Contains(it.Conflict, "by hand") || !strings.Contains(it.Diff, "-font-size = 20") {
		t.Errorf("file = %+v, want a conflict and its diff", it)
	}
}

func TestGhosttyReportsALinkThatLacksTheLine(t *testing.T) {
	m := effecttest.NewMachine()
	s := ghosttyStep(m, &effecttest.Out{})
	m.Files[ghosttyDir+"/config.ghostty"] = []byte("theme = mine\n")
	m.Links[ghosttyDir+"/config.ghostty"] = "/dots/config"
	it := reportOf(t, s)[ghosttyDir+"/config.ghostty"]
	if it.Found != "lacks it" || !strings.Contains(it.Conflict, "/dots/config") || it.Diff != "" {
		t.Errorf("config = %+v, want the line lacking, and the link named", it)
	}
}

func TestCldReportsItsVersion(t *testing.T) {
	s, m, _ := cldStep(true)
	items, err := s.Report(context.Background())
	if err != nil || items[0].Kind != step.PackageItem || items[0].Found != "installed 0.13.0" {
		t.Errorf("items = %+v, %v", items, err)
	}
	delete(m.Files, cldBin)
	delete(m.Modes, cldBin)
	if items, _ = s.Report(context.Background()); items[0].Found != "missing" {
		t.Errorf("items = %+v, want cld missing", items)
	}
}

func TestCableReportsItsAddress(t *testing.T) {
	const list = "header\nDock\n"
	m := effecttest.NewMachine()
	it := reportOf(t, cableStep(m, &effecttest.Out{}, list, cableSet))["Dock"]
	if it.Kind != step.ServiceItem || it.Found != it.Want {
		t.Errorf("set = %+v, want its address found", it)
	}
	it = reportOf(t, cableStep(m, &effecttest.Out{}, list, "DHCP Configuration\nIP address: 10.0.0.9\n"))["Dock"]
	if it.Found == it.Want || !strings.Contains(it.Found, "10.0.0.9") {
		t.Errorf("dhcp = %+v, want the address it has", it)
	}
	it = reportOf(t, cableStep(m, &effecttest.Out{}, "header\n*Dock\n", cableSet))["Dock"]
	if it.Found == it.Want || !strings.Contains(it.Conflict, "disabled") {
		t.Errorf("disabled = %+v, want a conflict: no run turns it on", it)
	}
	s := cableStep(m, &effecttest.Out{}, list, cableSet)
	s.Conf = conf.Conf{}
	if got := reportOf(t, s); len(got) != 0 {
		t.Errorf("items = %v, want none without a service", got)
	}
}

func TestTunnelReportsAMacThatIsDone(t *testing.T) {
	m := daemonMachine()
	got := reportOf(t, daemonStep(m))
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
	reportOf(t, daemonStep(m))
	if slices.Contains(m.Calls, "read root "+tunnelKey) || slices.Contains(m.Calls, "read "+tunnelKey) {
		t.Errorf("calls = %q, which read the private key", m.Calls)
	}
	asked := 0
	for _, c := range m.Calls {
		if strings.Contains(c, "mac tunnel key") {
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
	got := reportOf(t, daemonStep(m))
	for _, name := range []string{rootWs, "/h/.local/bin/ws"} {
		if it := got[name]; it.Found == it.Want || it.Diff != "" {
			t.Errorf("%s = %+v, want it to differ, with no diff", name, it)
		}
	}
}

func TestTunnelReportsADaemonLaunchdLacks(t *testing.T) {
	m := daemonMachine()
	m.Results["launchctl print system/"+daemonName] = effecttest.Result{Err: errors.New("no")}
	if it := reportOf(t, daemonStep(m))[daemonName]; it.Found != "not loaded" {
		t.Errorf("daemon = %+v", it)
	}
}

func TestTunnelWithoutAPeerReportsNoDaemon(t *testing.T) {
	got := reportOf(t, tunnelStep(effecttest.NewMachine()))
	if _, ok := got[daemonName]; ok {
		t.Errorf("items = %v, want no daemon before mac.conf names a peer", got)
	}
	if it := got[tunnelKey]; it.Kind != step.FileItem || it.Found != "missing" {
		t.Errorf("key = %+v", it)
	}
}
