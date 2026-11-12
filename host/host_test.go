package host_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/host"
)

type change struct {
	name string
	log  *[]string
	err  error
}

func (c change) String() string { return c.name }

func (c change) Apply(context.Context) error {
	*c.log = append(*c.log, c.name)
	return c.err
}

type part []host.Change

func (p part) Plan(context.Context) ([]host.Change, error) { return p, nil }

func TestGroupAppliesInOrder(t *testing.T) {
	var log []string
	g := host.Group{Title: "T", After: "A", Parts: []host.Resource{
		part{change{"one", &log, nil}, change{"two", &log, nil}}, part{change{"three", &log, nil}},
	}}
	p, err := g.Plan(context.Background())
	if err == nil {
		err = g.Apply(context.Background(), p)
	}
	if got := strings.Join(log, " "); err != nil || got != "one two three" {
		t.Errorf("applied %q, %v; want one two three", got, err)
	}
}

func TestGroupStopsAtAFailure(t *testing.T) {
	var log []string
	boom := errors.New("boom")
	p := step.Plan{Changes: []step.Change{change{"one", &log, boom}, change{"two", &log, nil}}}
	err := host.Group{}.Apply(context.Background(), p)
	if !errors.Is(err, boom) || !strings.HasPrefix(err.Error(), "one: ") || len(log) != 1 {
		t.Errorf("Apply = %v after %v; want one: boom, stopped", err, log)
	}
}

type forced struct{ change }

func (forced) Forced() bool { return true }

func TestGroupStillAppliesAForcedChange(t *testing.T) {
	var log []string
	boom := errors.New("boom")
	p := step.Plan{Changes: []step.Change{
		change{"one", &log, boom}, change{"two", &log, nil}, forced{change{"restart", &log, nil}}}}
	err := host.Group{}.Apply(context.Background(), p)
	if !errors.Is(err, boom) || strings.Join(log, " ") != "one restart" {
		t.Errorf("Apply = %v after %v; want boom after one restart", err, log)
	}
}
