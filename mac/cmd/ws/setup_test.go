package main

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
)

func TestSetupCommand(t *testing.T) {
	env := map[string]string{"HOME": "/h"}
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	tests := []struct {
		name       string
		args       []string
		wantStatus int
		wantErr    string
	}{
		{"unknown step", []string{"nope"}, 2, `unknown step "nope"`},
		{"two steps", []string{"ssh", "git"}, 2, "usage: ws setup"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			got := setupCommand(context.Background(), tt.args, 501, m, func(k string) string { return env[k] }, &bytes.Buffer{}, &stderr)
			if got != tt.wantStatus || !strings.Contains(stderr.String(), tt.wantErr) {
				t.Errorf("status = %d, stderr = %q; want %d, %q", got, stderr.String(), tt.wantStatus, tt.wantErr)
			}
		})
	}
	var stderr bytes.Buffer
	if got := setupCommand(context.Background(), []string{"ssh"}, 501, m, func(string) string { return "" }, &bytes.Buffer{}, &stderr); got != 1 ||
		!strings.Contains(stderr.String(), "is missing") {
		t.Errorf("without HOME: status = %d, stderr = %q; want a missing mac.conf", got, stderr.String())
	}
}

func TestSetupRefusesRootAndPrefixesWithWsProg(t *testing.T) {
	m := effecttest.NewMachine()
	env := map[string]string{"WS_PROG": "setup.sh"}
	var stderr bytes.Buffer
	if got := setupCommand(context.Background(), []string{"ssh"}, 0, m, func(k string) string { return env[k] }, &bytes.Buffer{}, &stderr); got != 1 ||
		!strings.HasPrefix(stderr.String(), "setup.sh: run it as yourself, not root") {
		t.Errorf("as root: status = %d, stderr = %q", got, stderr.String())
	}
	out := progOut{Out: &effecttest.Out{}, prog: "setup.sh", seen: map[string]bool{}}
	out.Say("%s: applying %d change(s)", "ssh", 1)
	out.Say("%s: %s", "cable", "would run x")
	out.Say("wrote %s", "f")
	out.Say("%s: %s", "ssh", setup.Write{Path: "/f", After: []byte("a\n")})
	out.Say("%s: %s", "ssh", setup.Backup{Path: "/f", To: "/f.~"})
	out.Warn("w %d", 1)
	out.Warn("w %d", 1)
	want := []string{
		"say: setup.sh: would run x", "say: setup.sh: wrote f",
		"say: --- /dev/null\n+++ /f\n@@ -0,0 +1 @@\n+a", "say: setup.sh: warning: w 1",
	}
	if got := out.Out.(*effecttest.Out).Lines; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
}

func TestSetupStepsComeInSetupShsOrder(t *testing.T) {
	var names []string
	for _, s := range setupSteps(conf.Conf{}, "/h", "", "/m", effecttest.NewMachine(), &effecttest.Out{}, &bytes.Buffer{}, false) {
		names = append(names, s.Name())
	}
	want := []string{"ssh", "cable", "git", "gateway", "ghostty", "cld", "tunnel", "networks"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("steps = %q; want %q", names, want)
	}
}

func TestSetupWithNoStepRunsThemAll(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	var stderr bytes.Buffer
	got := setupCommand(context.Background(), nil, 501, m, func(k string) string { return map[string]string{"HOME": "/h"}[k] },
		&bytes.Buffer{}, &stderr)
	if got == 2 || strings.Contains(stderr.String(), "usage:") {
		t.Errorf("status = %d, stderr = %q; want the steps run, not the usage", got, stderr.String())
	}
}

func TestSetupCheckWithNoStepPlansSSHOnce(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	for _, file := range []string{"new", "whole"} {
		for _, host := range []string{"ws", "ws-ssh"} {
			m.Results["ssh -G -F /tmp/x/"+file+" "+host] = effecttest.Result{}
		}
	}
	var stdout, stderr bytes.Buffer
	setupCommand(context.Background(), []string{"--check"}, 501, m, func(k string) string { return map[string]string{"HOME": "/h"}[k] },
		&stdout, &stderr)
	if n := strings.Count(stdout.String()+stderr.String(), "+++ /h/.ssh/config.d/ws"); n != 1 {
		t.Errorf("the config.d/ws diff prints %d times; want 1\n%s%s", n, &stdout, &stderr)
	}
}
