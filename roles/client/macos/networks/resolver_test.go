package networks_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/networks"
)

const resolverCmdHeader = "# Written by github.com/zadykian/ws: ws client macos resolver, which removes this file when given no address."

func resolverPlan(m *effecttest.Machine, domain string, addresses ...string) (string, error) {
	p, err := (&networks.Resolver{Domain: domain, Addresses: addresses, M: m, Out: &effecttest.Out{}}).Plan(context.Background())
	return fmt.Sprint(p.Changes), err
}

func TestResolverMakesTheDirectoryAndWritesTheFile(t *testing.T) {
	got, err := resolverPlan(effecttest.NewMachine(), "a.test", "10.0.0.1", "10.0.0.2")
	if err != nil || !strings.HasPrefix(got, "[create directory /etc/resolver") || !strings.Contains(got, "+nameserver 10.0.0.2") {
		t.Errorf("Plan = %q, %v", got, err)
	}
}

func TestResolverRefusesWhatTheShellRefused(t *testing.T) {
	for domain, addresses := range map[string][]string{"a b": nil, "a.test": {"10.0.0"},
		"b.test": {"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"}} {
		_, err := resolverPlan(effecttest.NewMachine(), domain, addresses...)
		if err == nil {
			t.Errorf("Plan of %q %q succeeded", domain, addresses)
		} else if strings.Contains(err.Error(), "ws client macos resolver") {
			t.Errorf("Plan of %q %q: %v names the command, which execute does", domain, addresses, err)
		}
	}
}

func TestResolverRemovesItsFileAndLeavesAnothers(t *testing.T) {
	m := effecttest.NewMachine()
	m.Root["/etc/resolver/a.test"], m.Root["/etc/resolver/b.test"] = effect.Attrs{Mode: 0o644}, effect.Attrs{Mode: 0o644}
	m.Files["/etc/resolver/a.test"] = []byte(resolverCmdHeader + "\nnameserver 10.0.0.1\n")
	m.Files["/etc/resolver/b.test"] = []byte("nameserver 10.0.0.1\n")
	if got, err := resolverPlan(m, "a.test"); err != nil || got != "[remove /etc/resolver/a.test]" {
		t.Errorf("Plan = %q, %v", got, err)
	}
	if got, err := resolverPlan(m, "b.test"); err != nil || got != "[]" {
		t.Errorf("Plan of another's file = %q, %v", got, err)
	}
	if _, err := resolverPlan(m, "b.test", "10.0.0.1"); err == nil {
		t.Error("Plan replaced another's file")
	}
}

func TestResolverAppliesAndSaysEachChange(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	r := &networks.Resolver{Domain: "a.test", Addresses: []string{"10.0.0.1"}, M: m, Out: out}
	p, err := r.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	want := []string{"say: create directory /etc/resolver, owner 0:0, mode 755"}
	if len(out.Lines) != 2 || out.Lines[0] != want[0] || !strings.HasPrefix(out.Lines[1], "say: ") {
		t.Errorf("out = %q; want the directory's line, then the file's", out.Lines)
	}
	if _, ok := m.Files["/etc/resolver/a.test"]; !ok {
		t.Error("Apply wrote no file")
	}
}
