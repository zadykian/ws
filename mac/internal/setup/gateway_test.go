package setup_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/setup"
)

func gatewayStep(m *effecttest.Machine, out *effecttest.Out) *setup.Gateway {
	return &setup.Gateway{
		Home: "/h", M: m, Out: out,
		LookPath: func(string) (string, error) { return "brew", nil },
	}
}

func TestGatewayAppThereNeedsNoBrew(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/h/Applications/JetBrains Gateway.app"] = 0o755
	p, err := gatewayStep(m, &effecttest.Out{}).Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || slices.ContainsFunc(m.Calls, func(c string) bool {
		return strings.HasPrefix(c, "run ")
	}) {
		t.Errorf("Plan = %v, %v, calls %v; want none", p.Changes, err, m.Calls)
	}
}

func TestGatewayInstallsTheCaskOnce(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["brew list --cask jetbrains-gateway"] = effecttest.Result{Err: errors.New("no")}
	m.Results["brew install --cask jetbrains-gateway"] = effecttest.Result{}
	s := gatewayStep(m, out)
	p, err := s.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("Plan = %v, %v; want one change", p.Changes, err)
	}
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
}

func TestGatewayFailedInstallIsAWarning(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["brew list --cask jetbrains-gateway"] = effecttest.Result{Err: errors.New("no")}
	m.Results["brew install --cask jetbrains-gateway"] = effecttest.Result{Err: errors.New("down")}
	s := gatewayStep(m, out)
	p, _ := s.Plan(context.Background())
	if err := s.Apply(context.Background(), p); err != nil {
		t.Fatalf("Apply = %v, want a warning", err)
	}
	if again, _ := s.Plan(context.Background()); len(again.Changes) != 0 {
		t.Errorf("plan after a failed install = %v, want none", again.Changes)
	}
	if !strings.Contains(strings.Join(out.Lines, "\n"), "the next run tries again") {
		t.Errorf("output %v lacks the warning", out.Lines)
	}
}

func TestGatewayWarnsWhereTheCaskHasNoApp(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["brew list --cask jetbrains-gateway"] = effecttest.Result{}
	p, err := gatewayStep(m, out).Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || len(out.Lines) != 1 {
		t.Errorf("Plan = %v, %v, output %v; want one warning", p.Changes, err, out.Lines)
	}
}
