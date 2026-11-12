package secrets

import (
	"strings"
	"testing"
)

func TestExpandChainsAndEscapes(t *testing.T) {
	s, err := Parse([]byte("[token]\ngithub = \"t\"\n[env]\nA = \"${env.B}-1\"\nB = \"x${token.github}$$y\"\n" +
		"[git.\"https://a.example\"]\ntoken = \"${env.A}\"\nusername = \"u$$\"\n[password]\nvpn_p12 = \"${env.B}\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	g := s.Git["https://a.example"]
	if s.Env["B"] != "xt$y" || s.Env["A"] != "xt$y-1" || g.Token != "xt$y-1" || g.Username != "u$" || s.Password.VPNP12 != "xt$y" {
		t.Errorf("got %+v", s)
	}
}

func TestExpandCopiesAndDecodeKeepsTheReferences(t *testing.T) {
	d, err := Decode([]byte("[token]\ngithub = \"t\"\n[env]\nA = \"${token.github}\"\n"))
	if err != nil {
		t.Fatal(err)
	}
	e, err := Expand(d)
	if err != nil || e.Env["A"] != "t" || d.Env["A"] != "${token.github}" {
		t.Errorf("expanded %+v, kept %+v, %v", e, d, err)
	}
}

func TestExpandNamesNamesAlone(t *testing.T) {
	for name, c := range map[string]struct{ in, want string }{
		"missing":  {"[env]\nA = \"${token.github}\"\n", "env.A refers to token.github, which is missing"},
		"kind":     {"[env]\nA = \"${nope.x}\"\n", "env.A refers to nope.x, which is missing"},
		"loop":     {"[env]\nA = \"${env.B}\"\nB = \"${env.A}\"\n", "env.A -> env.B -> env.A"},
		"self":     {"[env]\nA = \"s${env.A}\"\n", "env.A refers to itself"},
		"dollar":   {"[env]\nA = \"$\"\n", "env.A holds a $"},
		"git":      {"[git.\"https://a.example\"]\ntoken = \"${password.vpn_p12}\"\n", "git.https://a.example.token refers to password.vpn_p12"},
		"password": {"[password]\nvpn_p12 = \"${env.X}\"\n", "password.vpn_p12 refers to env.X"},
	} {
		_, err := Parse([]byte(c.in))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %q", name, err, c.want)
		}
	}
}
