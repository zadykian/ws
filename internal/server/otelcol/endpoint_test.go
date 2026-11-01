package otelcol_test

import (
	"strings"
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

func TestCheckEndpoint(t *testing.T) {
	for _, c := range []struct {
		url string
		ok  bool
	}{
		{"http://collector.example:4317", true},
		{"https://10.0.0.5:4318", true},
		{"https://[fd00::1]:4317", true},
		{"http://host_name:4317", true},
		{"collector:4317", false},
		{"http://collector", false},
		{"http://collector:0", false},
		{"http://collector:65535", true},
		{"http://collector:65536", false},
		{"http://collector:99999", false},
		{"http://collector:100000", false},
		{"http://collector:4317/path", false},
		{"HTTP://collector:4317", false},
		{"http://127.0.0.1:4317", false},
		{"http://127.0.0.1:4318", true},
		{"http://0.0.0.0:4317", false},
		{"http://[::]:4317", false},
		{"http://[0:0::0]:4317", false},
		{"http://[::ffff:127.0.0.1]:4317", false},
		{"http://localhost:4317", false},
		{"https://LocalHost.:4317", false},
		{"http://a.localhost:4317", false},
		{"http://notlocalhost:4317", true},
	} {
		err := otelcol.CheckEndpoint(c.url)
		if (err == nil) != c.ok {
			t.Errorf("CheckEndpoint(%q) = %v, want ok %v", c.url, err, c.ok)
		}
		if err != nil && strings.Contains(err.Error(), c.url) {
			t.Errorf("the error for %q holds the value: %v", c.url, err)
		}
	}
}
