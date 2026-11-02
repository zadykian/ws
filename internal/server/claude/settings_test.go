package claude

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const (
	cldBin   = "/root/.local/bin/cld"
	userArgs = "setup config user --permissions=read-only"
)

func newSettings(report string) (*effecttest.Machine, UserSettings) {
	m := effecttest.NewMachine()
	m.Files[cldBin] = nil
	m.Results["env HOME=/root mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["env HOME=/root rm -rf -- /tmp/x"] = effecttest.Result{}
	m.Results["env HOME=/root env CLAUDE_CONFIG_DIR=/tmp/x "+cldBin+" "+userArgs] = effecttest.Result{Out: []byte(report)}
	return m, UserSettings{M: m, User: root, Permissions: "read-only"}
}

func TestUserSettingsPlan(t *testing.T) {
	cases := []struct {
		name, report, want string
	}{
		{"left", "Left /tmp/x/settings.json as it is\n", ""},
		{"silent", "", ""},
		{"update", "Updated /tmp/x/settings.json: added model\n",
			"cld setup config user changes claude's user settings:\nUpdated /root/.claude/settings.json: added model"},
	}
	for _, c := range cases {
		m, s := newSettings(c.report)
		got, err := s.Plan(context.Background())
		switch {
		case err != nil:
			t.Errorf("%s: %v", c.name, err)
		case c.want == "" && len(got) != 0, c.want != "" && (len(got) != 1 || got[0].String() != c.want):
			t.Errorf("%s: %v, want %q", c.name, got, c.want)
		}
		if strings.Contains(strings.Join(m.Calls, "\n"), "CLAUDE_CONFIG_DIR=/root") {
			t.Errorf("%s: the plan ran cld on the real file", c.name)
		}
	}
}

func TestUserSettingsPlanCopiesTheFile(t *testing.T) {
	m, s := newSettings("Left it\n")
	m.Files["/root/.claude/settings.json"] = []byte("{}")
	m.Results["env HOME=/root cp -- /root/.claude/settings.json /tmp/x/settings.json"] = effecttest.Result{}
	if _, err := s.Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestUserSettingsRefusesADanglingLink(t *testing.T) {
	m, s := newSettings("Left it\n")
	m.Links["/root/.claude/settings.json"] = "/nowhere"
	if _, err := s.Plan(context.Background()); err == nil {
		t.Error("a link to no file is planned")
	}
}

func TestUserSettingsCleansUpAfterAFailedCopy(t *testing.T) {
	m, s := newSettings("")
	m.Files["/root/.claude/settings.json"] = []byte("{}")
	m.Results["env HOME=/root cp -- /root/.claude/settings.json /tmp/x/settings.json"] = effecttest.Result{Err: errors.New("denied")}
	if _, err := s.Plan(context.Background()); err == nil {
		t.Fatal("a failed copy is planned")
	}
	if last := m.Calls[len(m.Calls)-1]; last != "run env HOME=/root rm -rf -- /tmp/x" {
		t.Errorf("last call %q", last)
	}
}

func TestUserSettingsApplyAddsNotifications(t *testing.T) {
	m, s := newSettings("Created x\n")
	s.Notifications = "terminal_bell"
	run := "env HOME=/root env CLAUDE_CONFIG_DIR=/root/.claude " + cldBin + " " + userArgs + " --notifications=terminal_bell"
	m.Results["env HOME=/root env CLAUDE_CONFIG_DIR=/tmp/x "+cldBin+" "+userArgs+" --notifications=terminal_bell"] =
		effecttest.Result{Out: []byte("Created x\n")}
	m.Results[run] = effecttest.Result{}
	got, err := s.Plan(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("%v, %v", got, err)
	}
	if err := got[0].Apply(context.Background()); err != nil {
		t.Fatal(err)
	}
}
