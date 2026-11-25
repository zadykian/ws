package values

import (
	"bytes"
	"fmt"
	"reflect"
	"slices"

	"go.yaml.in/yaml/v3"
)

// YAML is the whole file for v: every key of Values, a table too, at its value in v and in the
// order of Values. The bootstrap writes it where a machine has none, so that each default is there
// to read and to edit. A list of plain values takes one line, the form that Set writes, and an
// unset pointer is null, as yaml would leave out a key that omitempty marks.
func (v Values) YAML() (string, error) { return yamlOf(v) }

// yamlOf is the whole file for v, a values type, as YAML gives it.
func yamlOf(v any) (string, error) {
	doc := &yaml.Node{Kind: yaml.MappingNode}
	all := reflect.ValueOf(v)
	for _, f := range fields(all.Type(), nil) {
		value := &yaml.Node{}
		if err := value.Encode(all.FieldByIndex(f.place).Interface()); err != nil {
			return "", fmt.Errorf("%s: %w", f.name, err)
		}
		flowLists(value)
		doc.Content = append(doc.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: f.name}, value)
	}
	text := bytes.NewBufferString("---\n")
	enc := yaml.NewEncoder(text)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
		return "", err
	}
	if err := enc.Close(); err != nil {
		return "", err
	}
	return text.String(), nil
}

// flowLists puts each list of plain values at node or below it on one line.
func flowLists(node *yaml.Node) {
	if node.Kind == yaml.SequenceNode && !slices.ContainsFunc(node.Content, func(n *yaml.Node) bool { return n.Kind != yaml.ScalarNode }) {
		node.Style = yaml.FlowStyle
	}
	for _, n := range node.Content {
		flowLists(n)
	}
}
