package resource_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/zadykian/ws/engine/core/resource"
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

func TestGroupJoinsTheReportsOfItsParts(t *testing.T) {
	g := resource.Group{Title: "T", Parts: []resource.Resource{
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
	g := resource.Group{Parts: []resource.Resource{
		reported{items: []step.Item{item("a")}}, reported{err: boom}, reported{items: []step.Item{item("b")}},
	}}
	got, err := g.Report(t.Context())
	if !errors.Is(err, boom) || !reflect.DeepEqual(got, []step.Item{item("a")}) {
		t.Errorf("items = %+v, err = %v, want a and the failure", got, err)
	}
}

func TestLazyReportsThePartsItMakes(t *testing.T) {
	l := resource.Lazy{Title: "L", Parts: func(context.Context) ([]resource.Resource, error) {
		return []resource.Resource{reported{items: []step.Item{item("a")}}}, nil
	}}
	got, err := step.Read(t.Context(), []step.Step{l})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got[0].Items, []step.Item{item("a")}) {
		t.Errorf("items = %+v, want the part's", got[0].Items)
	}
	l.Parts = func(context.Context) ([]resource.Resource, error) { return nil, errors.New("no disk") }
	if _, err := l.Report(t.Context()); err == nil {
		t.Error("Report = nil, want the failure to make the parts")
	}
}
