package roles

import "testing"

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
