package cmd

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// wsSecrets is a ws that takes the login as root, whose ws server ubuntu secrets runs here on ws,
// as root, or where ssh fails with err.
type wsSecrets struct {
	farWs
	ws  *effecttest.Machine
	err error
}

func (w wsSecrets) RunInput(ctx context.Context, in []byte, name string, args ...string) ([]byte, error) {
	at := slices.Index(args, "/usr/local/bin/ws")
	if name != "ssh" || at < 0 {
		return w.farWs.RunInput(ctx, in, name, args...)
	} else if w.err != nil {
		return nil, w.err
	}
	var far []string
	for _, a := range args[at+4:] {
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
	ws, err := r.reach(context.Background(), tui.Offscreen(p.machine))
	if err != nil {
		t.Fatal(err)
	}
	side := wsSide{ws}
	if err := side.Set(context.Background(), "env A", "dummy-value"); err != nil {
		t.Fatal(err)
	}
	held, err := side.Keys(context.Background())
	if s, lerr := secrets.Load(m, secrets.FilePath, id); err != nil || lerr != nil || s.Env["A"] != "dummy-value" ||
		!slices.Equal(held, []string{"env A"}) {
		t.Errorf("keys %q, file %+v, %v %v; want env A set", held, s, err, lerr)
	}
}

// openSecrets opens ws's secrets sheet of a Mac whose ws's file holds toml, and returns ws's machine.
func openSecrets(t *testing.T, toml string) (tui.Sheet, *effecttest.Machine, age.Identity, error) {
	m, id := credFixture()
	armored, err := secrets.Encrypt([]byte(toml), id.Recipient())
	if err != nil {
		t.Fatal(err)
	}
	m.Files[secrets.FilePath] = armored
	p, mac := macTerminal()
	p.machine = wsSecrets{farWs: newFarWs(mac, ""), ws: m}
	r := &varsReach{p: p}
	t.Cleanup(func() { r.close(context.Background()) })
	sheet, err := r.secretsSheet().Open(context.Background(), tui.Offscreen(r.p.machine))
	return sheet, m, id, err
}

func TestWsSecretsSheetSavesOnWs(t *testing.T) {
	sheet, m, id, err := openSecrets(t, "[token]\ngithub = \"dummy\"\n[git.\"https://h.example\"]\ntoken = \"dummy\"\nusername = \"ada\"\n")
	if err != nil {
		t.Fatal(err)
	}
	if specs := sheet.Fields(nil); len(specs) != 4 || specs[0].Help != "set: type a value to replace it" ||
		specs[3].Label != "git https://h.example" {
		t.Fatalf("fields = %+v", specs)
	}
	next, err := sheet.Save(context.Background(), nil, map[string]string{"token azure_devops": "dummy-new",
		"git https://h.example": "dummy-2"})
	got, lerr := secrets.Load(m, secrets.FilePath, id)
	if g := got.Git["https://h.example"]; err != nil || lerr != nil || got.Token.AzureDevOps != "dummy-new" ||
		got.Token.GitHub != "dummy" || g.Token != "dummy-2" || g.Username != "ada" {
		t.Errorf("file = %+v, %v %v; want the new tokens, and the rest kept", got, err, lerr)
	}
	if help := next.Fields(nil)[1].Help; !strings.HasPrefix(help, "set:") {
		t.Errorf("after the save: %q", help)
	}
	if _, err := next.Save(context.Background(), nil, map[string]string{"token nope": "dummy-secret"}); err == nil ||
		strings.Contains(err.Error(), "dummy-secret") {
		t.Errorf("err = %v, want a refusal with no value", err)
	}
}

func TestWsSecretsSheetWithoutTheCredential(t *testing.T) {
	p, m := macTerminal()
	p.machine = wsSecrets{farWs: newFarWs(m, ""), ws: effecttest.NewMachine()}
	r := &varsReach{p: p}
	defer r.close(context.Background())
	if _, err := r.secretsSheet().Open(context.Background(), tui.Offscreen(r.p.machine)); err == nil || !strings.Contains(err.Error(), "ws server ubuntu secrets init") {
		t.Errorf("err = %v, want one naming init", err)
	}
}
