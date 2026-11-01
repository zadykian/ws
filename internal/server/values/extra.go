package values

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Override returns data with each KEY=VALUE of pairs set over the file's own, as a run's -e does.
// A key that is not a value's is left out: the playbook's roles read it alone. A VALUE is read as
// YAML reads a plain scalar, so false is a boolean and 64 a number.
func Override(data []byte, pairs []string) ([]byte, error) {
	var doc yaml.Node
	if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	if doc.Kind == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}}
	}
	if doc.Kind != yaml.DocumentNode || doc.Content[0].Kind != yaml.MappingNode {
		return data, nil
	}
	known := keys(reflect.TypeFor[Values]())
	root := doc.Content[0]
	for _, p := range pairs {
		key, val, ok := strings.Cut(p, "=")
		if !ok || !known[key] {
			continue
		}
		node := &yaml.Node{Kind: yaml.ScalarNode, Value: val}
		set := false
		for i := 0; i+1 < len(root.Content); i += 2 {
			if root.Content[i].Value == key {
				root.Content[i+1], set = node, true
			}
		}
		if !set {
			root.Content = append(root.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, node)
		}
	}
	return yaml.Marshal(&doc)
}

// keys are the YAML keys of a struct, with the inlined structs' keys among them.
func keys(t reflect.Type) map[string]bool {
	out := map[string]bool{}
	for f := range t.Fields() {
		tag := f.Tag.Get("yaml")
		name, opts, _ := strings.Cut(tag, ",")
		switch {
		case opts == "inline":
			for k := range keys(f.Type) {
				out[k] = true
			}
		case name != "" && name != "-":
			out[name] = true
		}
	}
	return out
}
