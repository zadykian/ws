package ubuntu

import (
	"slices"
	"testing"
)

func TestListHasEachRoleOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range List() {
		if r.Name == "" || seen[r.Name] {
			t.Errorf("role %q is empty or twice in the list", r.Name)
		}
		seen[r.Name] = true
		if r.Steps == nil {
			t.Errorf("role %q has no steps", r.Name)
		}
	}
}

func TestListIsACopy(t *testing.T) {
	l := List()
	l[0].Name = "changed"
	if List()[0].Name == "changed" {
		t.Error("List returns the list itself")
	}
}

func TestZedRunsAfterFresh(t *testing.T) {
	var names []string
	for _, r := range List() {
		names = append(names, r.Name)
	}
	i := slices.Index(names, "zed")
	if i < 1 || names[i-1] != "fresh" {
		t.Errorf("zed is misplaced in %v", names)
	}
}

func TestFreshRunsJustBeforeZed(t *testing.T) {
	var names []string
	for _, r := range List() {
		names = append(names, r.Name)
	}
	i := slices.Index(names, "fresh")
	if i < 1 || names[i-1] != "otelcol" || names[i+1] != "zed" {
		t.Errorf("fresh is misplaced in %v", names)
	}
}
