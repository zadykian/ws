package setup_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
)

func resolverPlan(m *effecttest.Machine, domain string, addresses ...string) (string, error) {
	p, err := (&setup.Resolver{Domain: domain, Addresses: addresses, M: m, Out: &effecttest.Out{}}).Plan(context.Background())
	return fmt.Sprint(p.Changes), err
}

func TestResolverMakesTheDirectoryAndWritesTheFile(t *testing.T) {
	got, err := resolverPlan(effecttest.NewMachine(), "a.test", "10.0.0.1", "10.0.0.2")
	if err != nil || !strings.HasPrefix(got, "[would make /etc/resolver") || !strings.Contains(got, "+nameserver 10.0.0.2") {
		t.Errorf("Plan = %q, %v", got, err)
	}
}

func TestResolverRefusesWhatTheShellRefused(t *testing.T) {
	for domain, addresses := range map[string][]string{"a b": nil, "a.test": {"10.0.0"},
		"b.test": {"10.0.0.1", "10.0.0.2", "10.0.0.3", "10.0.0.4"}} {
		if _, err := resolverPlan(effecttest.NewMachine(), domain, addresses...); err == nil {
			t.Errorf("Plan of %q %q succeeded", domain, addresses)
		}
	}
}
