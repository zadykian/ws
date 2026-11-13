package setup_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
)

func TestGhosttyReportsAFreshMac(t *testing.T) {
	got := mactest.ReportOf(t, ghosttyStep(effecttest.NewMachine(), &effecttest.Out{}))
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
	it := mactest.ReportOf(t, s)[ghosttyDir+"/ws.ghostty"]
	if !strings.Contains(it.Conflict, "by hand") || !strings.Contains(it.Diff, "-font-size = 20") {
		t.Errorf("file = %+v, want a conflict and its diff", it)
	}
}

func TestGhosttyReportsALinkThatLacksTheLine(t *testing.T) {
	m := effecttest.NewMachine()
	s := ghosttyStep(m, &effecttest.Out{})
	m.Files[ghosttyDir+"/config.ghostty"] = []byte("theme = mine\n")
	m.Links[ghosttyDir+"/config.ghostty"] = "/dots/config"
	it := mactest.ReportOf(t, s)[ghosttyDir+"/config.ghostty"]
	if it.Found != "lacks it" || !strings.Contains(it.Conflict, "/dots/config") || it.Diff != "" {
		t.Errorf("config = %+v, want the line lacking, and the link named", it)
	}
}

// A report shows ws's lines alone of the user's Ghostty config.
func TestGhosttyReportShowsWsLinesAloneOfAUsersFile(t *testing.T) {
	m := effecttest.NewMachine()
	s := ghosttyStep(m, &effecttest.Out{})
	m.Files["/h/.config/ghostty/config"] = []byte(mactest.UsersText)
	mactest.CheckUsersFile(t, s, "+config-file = ?ws.ghostty")
}
