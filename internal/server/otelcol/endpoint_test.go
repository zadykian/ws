package otelcol_test

import (
	"testing"

	"github.com/zadykian/ws/internal/server/otelcol"
)

func TestEndpointReadsTheLastSettingWithoutSourcingTheFile(t *testing.T) {
	const name = "OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT"
	for _, c := range []struct{ env, want string }{
		{"", ""},
		{"OTHER=1\n", ""},
		{name + "=http://a:1\n", "http://a:1"},
		{"  export " + name + "=http://a:1  \n", "http://a:1"},
		{name + "=http://a:1\n" + name + "=http://b:2", "http://b:2"},
		{name + "=\"http://a:1\"\n", "http://a:1"},
		{name + "='http://a:1'\n", "http://a:1"},
		{name + "=\"http://a:1'\n", "\"http://a:1'"},
		{name + "=\"\"\n", ""},
		{name + "=$(touch /x)\n", "$(touch /x)"},
		{"#" + name + "=http://a:1\n", ""},
		{"x" + name + "=http://a:1\n", ""},
	} {
		if got := otelcol.Endpoint([]byte(c.env)); got != c.want {
			t.Errorf("Endpoint(%q) = %q, want %q", c.env, got, c.want)
		}
	}
}
