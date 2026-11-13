package ssh_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

func TestSSHReportsAFreshHome(t *testing.T) {
	got := mactest.ReportOf(t, sshStep(sshMachine(), &effecttest.Out{}))
	for name, want := range map[string]string{
		"/h/.ssh": "present", "/h/.ssh/config.d": "mode 700", "/h/.ssh/config.d/ws": "mode 600",
		"/h/.ssh/config": "holds Include config.d/ws",
	} {
		if got[name].Want != want || got[name].Found != "missing" || got[name].Conflict != "" {
			t.Errorf("%s = %+v, want %q and missing", name, got[name], want)
		}
	}
}

func TestSSHAgainReportsNothing(t *testing.T) {
	s := sshStep(sshMachine(), &effecttest.Out{})
	s.Again = true
	got, err := s.Report(context.Background())
	if err != nil || len(got) != 0 {
		t.Errorf("items = %v, %v, want none: the first run reports the same files", got, err)
	}
}

func TestSSHReportsAChangeToItsFile(t *testing.T) {
	const file = "/h/.ssh/config.d/ws"
	m := sshMachine()
	m.Dirs["/h/.ssh"], m.Dirs["/h/.ssh/config.d"] = 0o700, 0o755
	m.Files[file], m.Modes[file] = []byte(sshcfg.Header+"\nHost ws\n"), 0o600
	got := mactest.ReportOf(t, sshStep(m, &effecttest.Out{}))
	if it := got[file]; it.Found != "mode 600, other content" || !strings.Contains(it.Conflict, "by hand") ||
		!strings.Contains(it.Diff, "+    HostName") {
		t.Errorf("file = %+v, want a conflict and its diff", it)
	}
	if it := got["/h/.ssh/config.d"]; it.Found != "mode 755" || it.Conflict == "" {
		t.Errorf("directory = %+v, want its mode to stand against ws", it)
	}
}

func TestSSHReportLeavesTheUsersConfigOut(t *testing.T) {
	m := sshMachine()
	m.Files["/h/.ssh/config"] = []byte("Host ws\n    IdentityFile ~/.ssh/hidden\n")
	it := mactest.ReportOf(t, sshStep(m, &effecttest.Out{}))["/h/.ssh/config"]
	if it.Found != "lacks it" || !strings.Contains(it.Conflict, "line 1") || it.Diff != "" {
		t.Errorf("config = %+v, want the Include lacking, and line 1 named", it)
	}
	if strings.Contains(fmt.Sprintf("%+v", it), "hidden") {
		t.Errorf("config = %+v, which shows the user's file", it)
	}
}

func TestSSHReportShowsAnIncludeBelowASettingAsOutOfPlace(t *testing.T) {
	m := sshMachine()
	m.Files["/h/.ssh/config"] = []byte("Host a\n    User x\nInclude config.d/ws\n")
	items, err := sshStep(m, &effecttest.Out{}).Report(context.Background())
	i := slices.IndexFunc(items, func(it step.Item) bool { return it.Name == "/h/.ssh/config" })
	if err != nil || i < 0 || items[i].Found != "out of place" || !strings.Contains(items[i].Conflict, "below other settings") {
		t.Errorf("Report = %+v, %v; want the Include out of place", items, err)
	}
}

func TestSSHReportsWsCopyOfAKey(t *testing.T) {
	s := sshStep(mactest.AgentMachine(sshMachine(), mactest.AgentKey), &effecttest.Out{})
	s.Conf.IdentityPubKey = sshcfg.AgentKeys(mactest.AgentKey)[0].Pub()
	it := mactest.ReportOf(t, s)[mactest.CopyOf(mactest.AgentKey)]
	if it.Kind != step.FileItem || it.Want != "present" || it.Found != "missing" {
		t.Errorf("copy = %+v, want a file that is missing", it)
	}
}

// A report shows ws's lines alone of the user's ~/.ssh/config.
func TestSSHReportShowsWsLinesAloneOfAUsersFile(t *testing.T) {
	m := sshMachine()
	m.Files["/h/.ssh/config"] = []byte(mactest.UsersText)
	mactest.CheckUsersFile(t, sshStep(m, &effecttest.Out{}), "+Include config.d/ws")
}
