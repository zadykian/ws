package setup_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/setup"
)

const resolverCmdHeader = "# Written by github.com/zadykian/ws: ws mac resolver, which removes this file when given no address."

func resolverPlan(m *effecttest.Machine, domain string, addresses ...string) (string, error) {
	p, err := (&setup.Resolver{Domain: domain, Addresses: addresses, M: m, Out: &effecttest.Out{}}).Plan(context.Background())
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
		} else if strings.Contains(err.Error(), "ws mac resolver") {
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
