package otelcol_test

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/zadykian/ws/ubuntu/values"
)

// renderedSpec is units.toml.tmpl filled with the default values, as the check's bridge address
// needs.
func renderedSpec(t *testing.T) []byte {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := template.New("units.toml.tmpl").Parse(string(specText(t, "units.toml.tmpl")))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := tmpl.Execute(&out, v); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}
