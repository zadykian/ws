package main

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
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

// wsSecrets is a ws that takes the login as root, whose ws server secrets runs here on ws, as root.
type wsSecrets struct {
	farWs
	ws *effecttest.Machine
}

func (w wsSecrets) RunInput(ctx context.Context, in []byte, name string, args ...string) ([]byte, error) {
	at := slices.Index(args, "/usr/local/bin/ws")
	if name != "ssh" || at < 0 {
		return w.farWs.RunInput(ctx, in, name, args...)
	}
	var far []string
	for _, a := range args[at+3:] {
		far = append(far, strings.Trim(a, "'"))
	}
	status, out, errs := runServerSecrets(w.ws, 0, string(in), far...)
	if status != 0 {
		return nil, errors.New("[ssh]: exit status 1: \n" + errs)
	}
	return []byte(out), nil
}

// wsSide sets an entry on ws's file, the value on stdin, and lists the entries that keys prints.
func TestWsSideSetsAndListsOnWs(t *testing.T) {
	m, id := credFixture()
	p, mac := macTerminal()
	p.machine = wsSecrets{farWs: newFarWs(mac, ""), ws: m}
	r := &varsReach{p: p}
	defer r.close(context.Background())
	ws, err := r.reach(context.Background(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	side := wsSide{ws}
	if err := side.set(context.Background(), "env A", "dummy-value"); err != nil {
		t.Fatal(err)
	}
	held, err := side.keys(context.Background())
	if s, lerr := secrets.Load(m, serverSecrets, id); err != nil || lerr != nil || s.Env["A"] != "dummy-value" ||
		!slices.Equal(held, []string{"env A"}) {
		t.Errorf("keys %q, file %+v, %v %v; want env A set", held, s, err, lerr)
	}
}
