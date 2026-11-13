package engine

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
)

type textChange string

func (c textChange) String() string { return string(c) }

type kindedChange struct{ textChange }

func (kindedChange) Kind() step.ChangeKind { return step.Install }

type recorder []step.Event

func (r *recorder) Observe(e step.Event) { *r = append(*r, e) }

func TestJSONEventsPrintALinePerEventAndPassItOn(t *testing.T) {
	var b bytes.Buffer
	var next recorder
	j := newJSONEvents(&b, &next)
	events := []step.Event{
		{Kind: step.Planned, Parent: "base", Step: "Packages", Changes: []step.Change{kindedChange{"install git"}, textChange("note")}},
		{Kind: step.Started, Parent: "base", Step: "Packages", Index: 1},
		{Kind: step.Failed, Parent: "base", Step: "Packages", Index: 1, Err: errors.New("boom")},
		{Kind: step.Skipped, Parent: "base", Step: "Later", Index: 2, After: "Packages", Check: true},
	}
	for _, e := range events {
		j.Observe(e)
	}
	got := b.String()
	if lines := strings.Count(got, "\n"); lines != len(events) {
		t.Fatalf("got %d lines, want %d:\n%s", lines, len(events), got)
	}
	for _, frag := range []string{`"event":"failed","parent":"base","step":"Packages","index":1,"error":"boom"}`,
		`"event":"skipped","parent":"base","step":"Later","index":2,"check":true,"after":"Packages"}`,
		`"text":"install git"`, `"text":"note"`} {
		if !strings.Contains(got, frag) {
			t.Errorf("output lacks %s:\n%s", frag, got)
		}
	}
	if len(next) != len(events) {
		t.Errorf("next saw %d events, want %d", len(next), len(events))
	}
	if j.Err() != nil {
		t.Errorf("err = %v", j.Err())
	}
}
