package secrets

import (
	"slices"
	"strings"
	"testing"
)

func TestSetAndUnset(t *testing.T) {
	var s Secrets
	steps := []struct{ kind, name, value string }{
		{KindToken, "github", "dummy"}, {KindToken, "azure_devops", "dummy"}, {KindEnv, "MY_TOKEN", "dummy"},
		{KindPassword, "vpn_p12", "dummy"},
	}
	for _, st := range steps {
		if err := s.Set(st.kind, st.name, st.value); err != nil {
			t.Fatalf("Set(%s %s) = %v", st.kind, st.name, err)
		}
	}
	s.SetGit("https://git.example.com", Git{Token: "dummy"})
	if err := s.SetFile("vpn_sswan", []byte("dummy")); err != nil {
		t.Fatal(err)
	}
	want := []string{"env MY_TOKEN", "file vpn_sswan", "git https://git.example.com", "password vpn_p12",
		"token azure_devops", "token github"}
	if got := s.Keys(); !slices.Equal(got, want) {
		t.Fatalf("Keys = %q, want %q", got, want)
	}
	for _, k := range want {
		kind, name, _ := strings.Cut(k, " ")
		if err := s.Unset(kind, name); err != nil {
			t.Errorf("Unset(%s) = %v", k, err)
		}
	}
	if got := s.Keys(); len(got) != 0 {
		t.Errorf("Keys after Unset = %q, want none", got)
	}
}

func TestEditErrorsNameNoValue(t *testing.T) {
	var s Secrets
	tests := []struct {
		name string
		err  error
		want string
	}{
		{"unknown kind", s.Set("nope", "x", "dummy"), "no kind of entry"},
		{"unknown token", s.Set(KindToken, "x", "dummy"), "github or azure_devops"},
		{"unknown password", s.Set(KindPassword, "x", "dummy"), "want vpn_p12"},
		{"git by Set", s.Set(KindGit, "x", "dummy"), "git x"},
		{"unknown file", s.SetFile("x", []byte("dummy")), "vpn_p12 or vpn_sswan"},
		{"unset missing", s.Unset(KindEnv, "MY_TOKEN"), "env MY_TOKEN is not in the file"},
	}
	for _, tt := range tests {
		if tt.err == nil || !strings.Contains(tt.err.Error(), tt.want) || strings.Contains(tt.err.Error(), "dummy") {
			t.Errorf("%s: err = %v, want %q and no value", tt.name, tt.err, tt.want)
		}
	}
}
