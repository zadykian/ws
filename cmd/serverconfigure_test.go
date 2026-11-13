package cmd

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/tui"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

func TestServerConfigureRefusals(t *testing.T) {
	user, _ := wizardProcess("", false, "")
	user.Euid = 1000
	away, _ := wizardProcess("", false, "")
	plain, _ := wizardProcess("", false, "")
	plain = withEnv(plain, map[string]string{"TERM": "xterm"})
	plain.TTY, plain.ErrTTY = true, true
	for name, tc := range map[string]struct {
		p    cli.Env
		args []string
		want string
	}{
		"a user":      {user, []string{"server", "ubuntu", "configure"}, "by sudo"},
		"no terminal": {away, []string{"server", "ubuntu", "configure"}, "ws server ubuntu values set"},
		"--plain":     {plain, []string{"--plain", "server", "ubuntu", "configure"}, "ws server ubuntu values set"},
	} {
		status, stdout, stderr := runWs(tc.p, "", tc.args...)
		if status != 1 || stdout != "" || !strings.Contains(stderr, tc.want) || strings.Contains(stderr, enterScreen) {
			t.Errorf("%s: status = %d, stdout = %q, stderr = %q", name, status, stdout, stderr)
		}
	}
}

// The secrets sheet sets an entry of ws's file here and reads it as set, a git entry keeping its user.
func TestServerSecretsSheetSetsOnWs(t *testing.T) {
	ctx := context.Background()
	m, id := credFixture()
	p := fakeProcess()
	p.Machine, p.Euid = m, 0
	armored, err := secrets.Encrypt([]byte("[git.\"https://git.example\"]\nusername = \"ada\"\n"), id.Recipient())
	if err != nil {
		t.Fatal(err)
	}
	m.Files[secrets.FilePath] = armored
	sheet, err := serverSecretsSheet(p).Open(ctx, tui.Offscreen(p.Machine))
	if err != nil || !strings.HasPrefix(sheet.Fields(nil)[0].Help, "not set") {
		t.Fatalf("open: %v, fields %+v", err, sheet.Fields(nil))
	}
	sheet, err = sheet.Save(ctx, nil, map[string]string{"token github": "dummy-value", "git https://git.example": "dummy-token"})
	s, lerr := secrets.Load(m, secrets.FilePath, id)
	if err != nil || lerr != nil || s.Token.GitHub != "dummy-value" || s.Git["https://git.example"] !=
		(secrets.Git{Username: "ada", Token: "dummy-token"}) {
		t.Fatalf("save: %v, %v, %+v", err, lerr, s)
	}
	if got := sheet.Fields(nil)[0]; got.Label != "token github" || !strings.HasPrefix(got.Help, "set") {
		t.Errorf("after the save: %+v", got)
	}
}

// The host_vars sheet reads the file by the files script, run here as root with no ssh.
func TestServerVarsSheetReadsOnWs(t *testing.T) {
	p, m := wizardProcess("---\nws_user: ada\n", false, "")
	m.Results["env SUDO_USER=ada sh -s -- values"] = effecttest.Result{Out: []byte("ws_user: ada\n")}
	sheet, err := ubuntucli.HostVarsSheet(ubuntucli.ServerOpen).Open(context.Background(), tui.Offscreen(p.Machine))
	if err != nil {
		t.Fatal(err)
	}
	user := ""
	for _, f := range sheet.Fields(nil) {
		if f.Label == "ws_user" {
			user = f.Value
		}
	}
	if user != "ada" {
		t.Errorf("ws_user = %q of %+v", user, sheet.Fields(nil))
	}
}
