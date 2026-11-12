package host_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/step"
)

// reported is a part that says what it manages.
type reported struct {
	part
	items []step.Item
	err   error
}

func (r reported) Report(context.Context) ([]step.Item, error) { return r.items, r.err }

func item(name string) step.Item { return step.Item{Kind: step.FileItem, Name: name} }

// reportOf returns the one item of a resource's report.
func reportOf(t *testing.T, r step.Reporter) step.Item {
	t.Helper()
	items, err := r.Report(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %+v, want one", items)
	}
	return items[0]
}

func TestGroupJoinsTheReportsOfItsParts(t *testing.T) {
	g := host.Group{Title: "T", Parts: []host.Resource{
		reported{items: []step.Item{item("a")}}, part{}, reported{items: []step.Item{item("b")}},
	}}
	got, err := step.Read(t.Context(), []step.Step{g})
	if err != nil {
		t.Fatal(err)
	}
	if want := []step.Item{item("a"), item("b")}; !reflect.DeepEqual(got[0].Items, want) {
		t.Errorf("items = %+v, want %+v", got[0].Items, want)
	}
}

func TestGroupKeepsWhatItReadBeforeAFailure(t *testing.T) {
	boom := errors.New("boom")
	g := host.Group{Parts: []host.Resource{
		reported{items: []step.Item{item("a")}}, reported{err: boom}, reported{items: []step.Item{item("b")}},
	}}
	got, err := g.Report(t.Context())
	if !errors.Is(err, boom) || !reflect.DeepEqual(got, []step.Item{item("a")}) {
		t.Errorf("items = %+v, err = %v, want a and the failure", got, err)
	}
}

func TestLazyReportsThePartsItMakes(t *testing.T) {
	l := host.Lazy{Title: "L", Parts: func(context.Context) ([]host.Resource, error) {
		return []host.Resource{reported{items: []step.Item{item("a")}}}, nil
	}}
	got, err := step.Read(t.Context(), []step.Step{l})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got[0].Items, []step.Item{item("a")}) {
		t.Errorf("items = %+v, want the part's", got[0].Items)
	}
	l.Parts = func(context.Context) ([]host.Resource, error) { return nil, errors.New("no disk") }
	if _, err := l.Report(t.Context()); err == nil {
		t.Error("Report = nil, want the failure to make the parts")
	}
}
