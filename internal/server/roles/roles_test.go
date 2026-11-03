package roles

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

func TestZedRunsBetweenHelixAndJetbrains(t *testing.T) {
	var names []string
	for _, r := range List() {
		names = append(names, r.Name)
	}
	i := slices.Index(names, "zed")
	if i < 1 || names[i-1] != "helix" || names[i+1] != "jetbrains" {
		t.Errorf("zed is misplaced in %v", names)
	}
}
