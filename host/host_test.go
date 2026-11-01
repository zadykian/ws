package host_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/host"
)

type change struct {
	name string
	log  *[]string
}

func (c change) String() string { return c.name }

func (c change) Apply(context.Context) error {
	*c.log = append(*c.log, c.name)
	return nil
}

type part []host.Change

func (p part) Plan(context.Context) ([]host.Change, error) { return p, nil }

func TestGroupAppliesInOrder(t *testing.T) {
	var log []string
	g := host.Group{Title: "T", After: "A", Parts: []host.Resource{
		part{change{"one", &log}, change{"two", &log}}, part{change{"three", &log}},
	}}
	p, err := g.Plan(context.Background())
	if err == nil {
		err = g.Apply(context.Background(), p)
	}
	if got := strings.Join(log, " "); err != nil || got != "one two three" {
		t.Errorf("applied %q, %v; want one two three", got, err)
	}
}
