package otelcol_test

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/internal/server/otelcol"
	"go.yaml.in/yaml/v3"
)

func TestConfigHasClaudesPipelineOnlyWithAnEndpoint(t *testing.T) {
	for _, c := range []struct {
		endpoint        string
		remote, tlsFree bool
	}{{"", false, false}, {"https://c.example:4317", true, false}, {"http://c.example:4317", true, true}} {
		out, err := otelcol.Config(c.endpoint)
		if err != nil {
			t.Fatal(err)
		}
		var cfg map[string]any
		if err := yaml.Unmarshal(out, &cfg); err != nil {
			t.Fatalf("endpoint %q: the config is no YAML: %v", c.endpoint, err)
		}
		text := string(out)
		if got := strings.Contains(text, "metrics/claude:"); got != c.remote {
			t.Errorf("endpoint %q: claude's pipeline = %v, want %v", c.endpoint, got, c.remote)
		}
		if got := strings.Contains(text, "insecure: true"); got != c.tlsFree {
			t.Errorf("endpoint %q: insecure = %v, want %v", c.endpoint, got, c.tlsFree)
		}
		if strings.Contains(text, c.endpoint) && c.endpoint != "" {
			t.Errorf("endpoint %q: the config holds the secret value", c.endpoint)
		}
	}
}

func TestEnvNamesTheHomeAndHoldsTheEndpoint(t *testing.T) {
	out, err := otelcol.Env("/home/ws", "http://c.example:4317")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/home/ws/.secrets/env", "\nOTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT=http://c.example:4317\n"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("env = %q, want it to hold %q", out, want)
		}
	}
}
