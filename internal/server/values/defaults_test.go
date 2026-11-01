package values

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

// secretKeys are the role defaults that only the secrets file sets.
var secretKeys = []string{"shell_secret_env", "shell_git_credentials", "ikeward_p12_password"}

// TestDefaultsAreTheRolesDefaults checks that every key of a role's defaults/main.yml is a field, and a
// value with no template is the field's default. A role the playbook no longer holds is skipped.
func TestDefaultsAreTheRolesDefaults(t *testing.T) {
	files, err := filepath.Glob("../../../roles/*/defaults/main.yml")
	if err != nil {
		t.Fatal(err)
	}
	def := Default()
	def.derive()
	rv := reflect.ValueOf(def)
	known := tags(rv.Type())
	for _, file := range files {
		data, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			t.Fatal(err)
		}
		var raw map[string]yaml.Node
		if err := yaml.Unmarshal(data, &raw); err != nil {
			t.Fatal(err)
		}
		for key, node := range raw {
			if contains(secretKeys, key) {
				continue // the secrets file's keys: the values file refuses them
			}
			if !contains(known, key) {
				t.Errorf("%s: %s is no field", file, key)
				continue
			}
			if strings.Contains(node.Value, "{{") {
				continue
			}
			var want any
			if err := node.Decode(&want); err != nil {
				t.Fatal(err)
			}
			got := fieldByTag(rv, key)
			var back any
			enc, err := yaml.Marshal(got.Interface())
			if err != nil {
				t.Fatal(err)
			}
			if err := yaml.Unmarshal(enc, &back); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(normalize(want), normalize(back)) {
				t.Errorf("%s: %s is %v, the default is %v", file, key, got.Interface(), want)
			}
		}
	}
}
