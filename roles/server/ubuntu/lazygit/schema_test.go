package lazygit

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
)

// lazygitSchema compiles the schema that LAZYGIT_SCHEMA names, which make gotest sets from
// tools/run. A go test of its own has none, and skips.
func lazygitSchema(t *testing.T) *jsonschema.Schema {
	path := os.Getenv("LAZYGIT_SCHEMA")
	if path == "" {
		t.Skip("LAZYGIT_SCHEMA is unset: make gotest sets it to lazygit's schema")
	}
	schema, err := jsonschema.NewCompiler().Compile(path)
	if err != nil {
		t.Fatalf("lazygit's schema: %v", err)
	}
	return schema
}

// instance reads a config as the schema's validator takes it: YAML, through JSON.
func instance(t *testing.T, cfg []byte) any {
	var doc any
	if err := yaml.Unmarshal(cfg, &doc); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	v, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// lazygit loads a key it lacks, or a value of another kind, with no error. So the schema of the
// version that Ubuntu's archive holds is the only check of the config.
func TestTheConfigMeetsLazygitsSchema(t *testing.T) {
	schema := lazygitSchema(t)
	if err := schema.Validate(instance(t, config)); err != nil {
		t.Errorf("config.yml: %v", err)
	}
	misspelt := bytes.Replace(config, []byte("method: never"), []byte("methd: never"), 1)
	if schema.Validate(instance(t, misspelt)) == nil {
		t.Error("the schema takes a key that lazygit lacks")
	}
}
