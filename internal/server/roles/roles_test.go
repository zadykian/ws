package roles

import (
	"os"
	"slices"
	"testing"

	"go.yaml.in/yaml/v3"
)

// siteRoles returns the names of the roles that site.yml lists, in its order.
func siteRoles(t *testing.T) []string {
	data, err := os.ReadFile("../../../site.yml")
	if err != nil {
		t.Fatal(err)
	}
	var plays []struct {
		Roles []struct {
			Role string   `yaml:"role"`
			Tags []string `yaml:"tags"`
		} `yaml:"roles"`
	}
	if err := yaml.Unmarshal(data, &plays); err != nil {
		t.Fatal(err)
	}
	if len(plays) != 1 {
		t.Fatalf("site.yml has %d plays, want 1", len(plays))
	}
	var names []string
	for _, r := range plays[0].Roles {
		if !slices.Equal(r.Tags, []string{r.Role}) {
			t.Errorf("role %s has tags %v, want its name alone", r.Role, r.Tags)
		}
		names = append(names, r.Role)
	}
	return names
}

func TestPlaybookRolesAreSiteYml(t *testing.T) {
	got, want := Playbook(), siteRoles(t)
	if !slices.Equal(got, want) {
		t.Errorf("the playbook's roles are %v, site.yml lists %v", got, want)
	}
}

func TestListHasEachRoleOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range List() {
		if r.Name == "" || seen[r.Name] {
			t.Errorf("role %q is empty or twice in the list", r.Name)
		}
		seen[r.Name] = true
	}
}

func TestListIsACopy(t *testing.T) {
	l := List()
	l[0].Name = "changed"
	if List()[0].Name == "changed" {
		t.Error("List returns the list itself")
	}
}
