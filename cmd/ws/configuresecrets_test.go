package main

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/tui"
)

func TestSecretsSheetSaysSetOrNotAndNeverShowsAValue(t *testing.T) {
	m, id := editFixture(t, "[token]\ngithub = \"dummy-token\"\n[env]\nA = \"dummy-env\"\n[git]\n")
	p := withEnv(fakeProcess(), map[string]string{"HOME": home, "USER": "me"})
	p.machine = m
	store := newSecretsStore(p)
	held, err := heldSecrets(context.Background(), store)
	if err != nil {
		t.Fatal(err)
	}
	specs := secretsSheet(storeSide{store}, held).Fields(nil)
	want := map[string]string{
		"token github": "set", "token azure_devops": "not set", "password vpn_p12": "not set", "env A": "set",
	}
	if len(specs) != len(want) {
		t.Fatalf("fields = %+v, want %d", specs, len(want))
	}
	for _, s := range specs {
		if s.Kind != tui.PasswordField || s.Value != "" || strings.Contains(s.Help, "dummy") ||
			!strings.HasPrefix(s.Help, want[s.Label]+":") {
			t.Errorf("field %+v, want a password that says %q", s, want[s.Label])
		}
	}
	// A save writes the edits and keeps the rest, then the sheet says set.
	next, err := secretsSheet(storeSide{store}, held).Save(context.Background(), nil, map[string]string{"token azure_devops": "dummy-new"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := secrets.LoadRaw(m, file, id)
	if err != nil || got.Token.AzureDevOps != "dummy-new" || got.Token.GitHub != "dummy-token" || got.Env["A"] != "dummy-env" {
		t.Errorf("file = %+v, %v; want the new token and the old entries", got, err)
	}
	if help := next.Fields(nil)[1].Help; !strings.HasPrefix(help, "set:") {
		t.Errorf("after the save: %q", help)
	}
}

func TestSecretsSheetKeepsAGitUserName(t *testing.T) {
	m, id := editFixture(t, "[git.\"https://h.example\"]\ntoken = \"dummy\"\nusername = \"ada\"\n")
	p := withEnv(fakeProcess(), map[string]string{"HOME": home, "USER": "me"})
	p.machine = m
	store := newSecretsStore(p)
	if _, err := secretsSheet(storeSide{store}, nil).Save(context.Background(), nil, map[string]string{"git https://h.example": "dummy-2"}); err != nil {
		t.Fatal(err)
	}
	got, err := secrets.LoadRaw(m, file, id)
	if g := got.Git["https://h.example"]; err != nil || g.Token != "dummy-2" || g.Username != "ada" {
		t.Errorf("git = %+v, %v; want the new token and the user name", g, err)
	}
}

func TestSecretsSheetSaveRefusalNamesNoValue(t *testing.T) {
	m, _ := editFixture(t, "")
	p := withEnv(fakeProcess(), map[string]string{"HOME": home, "USER": "me"})
	p.machine = m
	_, err := secretsSheet(storeSide{newSecretsStore(p)}, nil).Save(context.Background(), nil, map[string]string{"token nope": "dummy-secret"})
	if err == nil || strings.Contains(err.Error(), "dummy-secret") {
		t.Errorf("err = %v, want a refusal with no value", err)
	}
}

func TestHeldSecretsOfNoItem(t *testing.T) {
	p, _ := macTerminal()
	held, err := heldSecrets(context.Background(), newSecretsStore(p))
	if err != nil || held != nil {
		t.Errorf("held = %v, %v; want none and no error", held, err)
	}
}
