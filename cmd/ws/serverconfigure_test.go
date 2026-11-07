package main

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/secrets"
)

// The secrets sheet sets an entry of ws's file here and reads it as set, a git entry keeping its user.
func TestServerSecretsSheetSetsOnWs(t *testing.T) {
	ctx := context.Background()
	m, id := credFixture()
	p := fakeProcess()
	p.machine, p.euid = m, 0
	store, _ := serverSecretsStore(p)
	held := secrets.Secrets{Git: map[string]secrets.Git{"https://git.example": {Username: "ada"}}}
	if err := store.write(held, id); err != nil {
		t.Fatal(err)
	}
	sheet, err := serverSecretsSheet(p).Open(ctx, nil)
	if err != nil || !strings.HasPrefix(sheet.Fields(nil)[0].Help, "not set") {
		t.Fatalf("open: %v, fields %+v", err, sheet.Fields(nil))
	}
	sheet, err = sheet.Save(ctx, nil, map[string]string{"token github": "dummy-value", "git https://git.example": "dummy-token"})
	s, lerr := secrets.Load(m, serverSecrets, id)
	if err != nil || lerr != nil || s.Token.GitHub != "dummy-value" || s.Git["https://git.example"] !=
		(secrets.Git{Username: "ada", Token: "dummy-token"}) {
		t.Fatalf("save: %v, %v, %+v", err, lerr, s)
	}
	if got := sheet.Fields(nil)[0]; got.Label != "token github" || !strings.HasPrefix(got.Help, "set") {
		t.Errorf("after the save: %+v", got)
	}
}
