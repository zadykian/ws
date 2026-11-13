package cmd

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/effect/units"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
)

func TestServerRefusal(t *testing.T) {
	ok := []byte("NAME=\"Ubuntu\"\nID=ubuntu\nVERSION_ID=\"26.04\"\n")
	for _, tc := range []struct {
		name string
		euid int
		text string
		want string
	}{
		{"ubuntu 26.04 as root", 0, string(ok), ""},
		{"not root", 1000, string(ok), "run it as root"},
		{"older ubuntu", 0, "ID=ubuntu\nVERSION_ID=\"24.04\"\n", "Ubuntu 26.04 is the one system for this, not ubuntu 24.04"},
		{"debian", 0, "ID=debian\nVERSION_ID=\"13\"\n", "Ubuntu 26.04 is the one system for this, not debian 13"},
		{"no file", 0, "", "Ubuntu 26.04 is the one system for this, not  "},
		{"quoted id", 0, "ID=\"ubuntu\"\nVERSION_ID=26.04\n", ""},
	} {
		if got := serverRefusal(tc.euid, []byte(tc.text)); got != tc.want {
			t.Errorf("%s: serverRefusal = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestServerApplyRefusesOtherArgs(t *testing.T) {
	for _, args := range [][]string{
		{"apply", "--tags", "base"}, {"apply", "--only", "nothing"}, {"apply", "--only", "nothing/step"},
		{"apply", "--only", "base,"}, {"apply", "--only", ""},
		{"apply", "--only"}, {"apply", "-C"}, {"apply", "extra"}, {"other"},
	} {
		status, _, stderr := runWs(fakeProcess(), "", append([]string{"server", "ubuntu"}, args...)...)
		if status != 2 || !strings.Contains(stderr, "--help' for usage.") {
			t.Errorf("%q: status = %d, stderr = %q", args, status, stderr)
		}
	}
}

func TestServerApplyRefusesNonRoot(t *testing.T) {
	p := fakeProcess()
	m := p.Machine.(*effecttest.Machine)
	p.Euid = 1000
	m.Files["/etc/os-release"] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	status, _, stderr := runWs(p, "", "server", "ubuntu", "apply")
	if status != 1 || stderr != "ws server ubuntu apply: run it as root\n" {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "exec ") }) {
		t.Errorf("calls = %q, want no exec", m.Calls)
	}
}

func TestServerApplyHelp(t *testing.T) {
	status, stdout, _ := runWs(fakeProcess(), "", "server", "ubuntu", "apply", "-h")
	for _, want := range []string{"--check", "--only", "ROLE/STEP", "--secrets", "ws-secrets"} {
		if status != 0 || !strings.Contains(stdout, want) {
			t.Errorf("status = %d, help = %q, want it to hold %q", status, stdout, want)
		}
	}
}

func TestLoadSecrets(t *testing.T) {
	id, _ := age.GenerateX25519Identity()
	m := effecttest.NewMachine()
	var buf bytes.Buffer
	a := armor.NewWriter(&buf)
	w, _ := age.Encrypt(a, id.Recipient())
	_, _ = w.Write([]byte("[env]\nA = \"hunter2\"\n"))
	_ = w.Close()
	_ = a.Close()
	m.Files["/s.age"] = buf.Bytes()
	env := func(k string) string {
		if k == "CREDENTIALS_DIRECTORY" {
			return "/creds"
		}
		return ""
	}
	m.Files["/creds/ws-secrets"] = []byte("# created: now\n" + id.String() + "\n")
	s, err := loadSecrets(m, "/s.age", env)
	if err != nil || s.Env["A"] != "hunter2" || !s.Whole {
		t.Errorf("got %+v, %v", s, err)
	}
	if s, err = loadSecrets(m, "", noEnv); err != nil || s.Env != nil || s.Whole {
		t.Errorf("no file: %+v, %v", s, err)
	}
	other, _ := age.GenerateX25519Identity()
	m.Files["/creds/ws-secrets"] = []byte(other.String())
	if _, err = loadSecrets(m, "/s.age", env); err == nil || strings.Contains(err.Error(), "hunter2") {
		t.Errorf("wrong identity: %v", err)
	}
	if _, err = loadSecrets(m, "/s.age", noEnv); err == nil || !strings.Contains(err.Error(), "LoadCredential") {
		t.Errorf("no credential: %v", err)
	}
	delete(m.Files, "/creds/ws-secrets")
	if _, err = loadSecrets(m, "/s.age", env); err == nil {
		t.Error("no credential file: no error")
	}
}

func TestServerFansRefusesAConfigThatFailsItsCheck(t *testing.T) {
	p := fakeProcess()
	p.Machine.(*effecttest.Machine).Files["/ws-fans.json"] = []byte(`{"interval": 0}`)
	status, _, stderr := runWs(p, "", "server", "ubuntu", "fans", "/ws-fans.json")
	if status != 1 || !strings.HasPrefix(stderr, "ws server ubuntu fans: /ws-fans.json: ") || !strings.Contains(stderr, "interval") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
	if status, _, _ = runWs(p, "", "server", "ubuntu", "fans", "a", "b"); status != 2 {
		t.Errorf("two configs: status = %d, want 2", status)
	}
	if status, _, stderr = runWs(p, "", "server", "ubuntu", "fans"); status != 1 || !strings.Contains(stderr, "/etc/ws-fans.json") {
		t.Errorf("no config: status = %d, stderr = %q", status, stderr)
	}
}

func noEnv(string) string { return "" }

func TestServerStateRefusesNonRootAndOtherArgs(t *testing.T) {
	p := fakeProcess()
	m := p.Machine.(*effecttest.Machine)
	p.Euid = 1000
	m.Files["/etc/os-release"] = []byte("ID=ubuntu\nVERSION_ID=\"26.04\"\n")
	status, stdout, stderr := runWs(p, "", "server", "ubuntu", "state", "--json")
	if status != 1 || stdout != "" || stderr != "ws server ubuntu state: run it as root\n" {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	for _, args := range [][]string{{"state", "extra"}, {"state", "--only", "nothing"}, {"state", "--check"},
		{"state", "--secrets", "f"}} {
		status, _, stderr := runWs(fakeProcess(), "", append([]string{"server", "ubuntu"}, args...)...)
		if status != 2 || !strings.Contains(stderr, "--help' for usage.") {
			t.Errorf("%q: status = %d, stderr = %q", args, status, stderr)
		}
	}
}

func TestServerStateRefusesAStepItsRoleLacks(t *testing.T) {
	p, _ := serverTerminal()
	status, stdout, stderr := runWs(p, "", "server", "ubuntu", "state", "--only", "tmux/no such")
	if status != 2 || stdout != "" || !strings.Contains(stderr, `no step "no such" in tmux`) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestServerStateHelp(t *testing.T) {
	status, stdout, _ := runWs(fakeProcess(), "", "server", "ubuntu", "state", "-h")
	for _, want := range []string{"--json", "--only", "applies nothing", "no secrets"} {
		if status != 0 || !strings.Contains(stdout, want) {
			t.Errorf("status = %d, help = %q, want it to hold %q", status, stdout, want)
		}
	}
}

func TestWriteState(t *testing.T) {
	states := []role.RoleState{{Role: "base", Steps: []step.Report{
		{Step: "Packages", Items: []step.Item{{Kind: step.PackageItem, Name: "git", Want: "installed", Found: "missing"}},
			Changes: []step.Pending{{Kind: step.Install, Text: "install git"}}},
		{Step: "Config", Items: []step.Item{{Kind: step.FileItem, Name: "/etc/x", Want: "written", Found: "differs",
			Conflict: "changed by hand", Diff: "-a\n+b\n"}}, Changes: []step.Pending{{Text: "write /etc/x\n-a\n+b\n"}}},
		{Step: "Later", After: "Config"},
	}}}
	var b bytes.Buffer
	if err := writeState(&b, states); err != nil {
		t.Fatal(err)
	}
	want := `base
  Packages
    package git: found missing, want installed
    pending: install git
  Config
    file /etc/x: found differs, want written
      conflict: changed by hand
      -a
      +b
    pending: write /etc/x
  Later
    not read: after Config
`
	if b.String() != want {
		t.Errorf("got\n%s\nwant\n%s", b.String(), want)
	}
}

func TestRoleSectionsNameTheRoles(t *testing.T) {
	reports := []step.Report{{Step: "Packages"}}
	got := roleSections([]role.RoleState{{Role: "base", Steps: reports}, {Role: "docker"}})
	if len(got) != 2 || got[0].Name != "base" || len(got[0].Reports) != 1 || got[0].Reports[0].Step != "Packages" ||
		got[1].Name != "docker" {
		t.Errorf("sections = %+v", got)
	}
}

func TestServerStateDrawsTheScreenAtATerminal(t *testing.T) {
	p, _ := serverTerminal()
	p.OutTTY = true
	status, stdout, stderr := runWs(p, "q", "server", "ubuntu", "state", "--only", "tmux")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	status, stdout, stderr = runWs(p, "", "server", "ubuntu", "state", "--json", "--only", "tmux")
	if status != 0 || !strings.Contains(stdout, `"role": "tmux"`) || strings.Contains(stderr, enterScreen) {
		t.Errorf("--json: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestPrepareServerGivesTheRolesTheMachinesUnits(t *testing.T) {
	p, _ := serverTerminal()
	c := &cobra.Command{Use: "apply"}
	c.SetContext(t.Context())
	r, err := prepareServer(c, p, applyOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.cfg.U.(units.OSUnits); !ok {
		t.Errorf("units = %T, want units.OSUnits", r.cfg.U)
	}
}
