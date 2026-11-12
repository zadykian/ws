package main

import (
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
)

// editFixture is ws's credential with an identity, and a file of its own TOML.
func editFixture(t *testing.T, toml string) (*effecttest.Machine, *age.X25519Identity) {
	t.Helper()
	m, id := credFixture()
	if toml != "" {
		armored, err := secrets.Encrypt([]byte(toml), id.Recipient())
		if err != nil {
			t.Fatal(err)
		}
		m.Files[secrets.FilePath] = armored
	}
	return m, id
}

func runSecrets(m *effecttest.Machine, stdin string, args ...string) (int, string, string) {
	return runServerSecrets(m, 0, stdin, args...)
}

func TestSecretsKeysAndUnset(t *testing.T) {
	m, id := editFixture(t, "[token]\ngithub = \"dummy\"\n[env]\nB = \"dummy\"\nA = \"${token.github}\"\n")
	status, out, _ := runSecrets(m, "", "keys")
	if status != 0 || out != "env A\nenv B\ntoken github\n" {
		t.Errorf("keys = %d %q, want the entries sorted", status, out)
	}
	if strings.Contains(out, "dummy") {
		t.Error("keys printed a value")
	}
	if status, _, errs := runSecrets(m, "", "unset", "env", "B"); status != 0 {
		t.Fatalf("unset: status %d, stderr %q", status, errs)
	}
	s, err := secrets.Load(m, secrets.FilePath, id)
	if err != nil || len(s.Env) != 1 || s.Env["A"] != "dummy" {
		t.Errorf("after unset: %+v, %v; want A alone", s, err)
	}
	status, _, errs := runSecrets(m, "", "unset", "env", "B")
	if status != 1 || !strings.Contains(errs, "env B is not in the file") {
		t.Errorf("unset again: status %d, stderr %q", status, errs)
	}
}

func TestSecretsSetKeysUnset(t *testing.T) {
	m, id := editFixture(t, "[env]\nOLD = \"x\"\n")
	m.Files["/p12"] = []byte{0, 1, 2}
	steps := [][]string{{"set", "token", "github"}, {"set", "env", "MY_TOKEN"}, {"set", "password", "vpn_p12"},
		{"set", "git", "https://git.example.com", "--username", "bob", "--proactive-auth"},
		{"set", "file", "vpn_p12", "/p12"}}
	for _, a := range steps {
		if status, _, errs := runSecrets(m, "dummy-value\n", a...); status != 0 {
			t.Fatalf("%q: status %d, stderr %q", a, status, errs)
		}
	}
	s, err := secrets.Load(m, secrets.FilePath, id)
	if err != nil {
		t.Fatal(err)
	}
	g := s.Git["https://git.example.com"]
	if s.Token.GitHub != "dummy-value" || s.Env["MY_TOKEN"] != "dummy-value" || s.Password.VPNP12 != "dummy-value" ||
		g.Token != "dummy-value" || g.Username != "bob" || !g.ProactiveAuth || len(s.File.VPNP12) != 3 {
		t.Errorf("secrets = %+v, want what was set", s)
	}
	if m.Modes[secrets.FilePath] != 0o600 {
		t.Errorf("mode %o, want 600", m.Modes[secrets.FilePath])
	}
	status, out, _ := runSecrets(m, "", "keys")
	want := "env MY_TOKEN\nenv OLD\nfile vpn_p12\ngit https://git.example.com\npassword vpn_p12\ntoken github\n"
	if status != 0 || out != want {
		t.Errorf("keys = %d %q, want %q", status, out, want)
	}
	if strings.Contains(out, "dummy-value") {
		t.Error("keys printed a value")
	}
	if status, _, errs := runSecrets(m, "", "unset", "env", "OLD"); status != 0 {
		t.Fatalf("unset: status %d, stderr %q", status, errs)
	}
	if _, out, _ = runSecrets(m, "", "keys"); strings.Contains(out, "env OLD") {
		t.Errorf("keys = %q, want OLD dropped", out)
	}
}

func TestSecretsSetFileFromStdin(t *testing.T) {
	m, id := editFixture(t, "")
	if status, _, errs := runSecrets(m, "conn ws\n", "set", "file", "vpn_sswan"); status != 0 {
		t.Fatalf("status %d, stderr %q", status, errs)
	}
	if s, err := secrets.Load(m, secrets.FilePath, id); err != nil || string(s.File.VPNSswan) != "conn ws\n" {
		t.Errorf("vpn_sswan = %q, %v; want stdin whole", s.File.VPNSswan, err)
	}
}

func TestSecretsValueFromStdin(t *testing.T) {
	m, id := editFixture(t, "")
	for in, want := range map[string]string{"dummy\n": "dummy", "dummy": "dummy", "dummy\n\n": "dummy\n", "a\r\n": "a\r"} {
		if status, _, errs := runSecrets(m, in, "set", "env", "A"); status != 0 {
			t.Fatalf("%q: status %d, stderr %q", in, status, errs)
		}
		s, _ := secrets.Load(m, secrets.FilePath, id)
		if s.Env["A"] != want {
			t.Errorf("stdin %q gave %q, want %q", in, s.Env["A"], want)
		}
	}
}

func TestSecretsSetRefuses(t *testing.T) {
	tests := []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{"env name", "dummy\n", []string{"set", "env", "A-B"}, "env"},
		{"empty value", "\n", []string{"set", "env", "A"}, "is empty"},
		{"unknown token", "dummy\n", []string{"set", "token", "nope"}, "token nope"},
		{"git URL", "dummy\n", []string{"set", "git", "http://git.example.com"}, "https://"},
		{"missing file", "", []string{"set", "file", "vpn_p12", "/none"}, "/none"},
		{"empty stdin", "", []string{"set", "file", "vpn_p12"}, "vpn_p12 on stdin is empty"},
		{"unset missing", "", []string{"unset", "env", "A"}, "env A is not in the file"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, _ := editFixture(t, "")
			before := len(m.Files)
			status, _, errs := runSecrets(m, tt.stdin, tt.args...)
			if status != 1 || !strings.Contains(errs, tt.want) || strings.Contains(errs, "dummy") {
				t.Errorf("status %d, stderr %q; want 1 and %q with no value", status, errs, tt.want)
			}
			if len(m.Files) != before {
				t.Error("a refused edit wrote the file")
			}
		})
	}
}

func TestSecretsSetGitKeepsWhatAFlagLeavesOut(t *testing.T) {
	m, id := editFixture(t, "[git.\"https://h.example\"]\ntoken = \"old\"\nusername = \"bob\"\nproactive_auth = true\n")
	if status, _, errs := runSecrets(m, "dummy-2\n", "set", "git", "https://h.example"); status != 0 {
		t.Fatalf("status %d, stderr %q", status, errs)
	}
	s, _ := secrets.Load(m, secrets.FilePath, id)
	if g := s.Git["https://h.example"]; g.Token != "dummy-2" || g.Username != "bob" || !g.ProactiveAuth {
		t.Errorf("git = %+v, want the new token with bob and proactive_auth", g)
	}
}
