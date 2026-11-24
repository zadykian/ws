package otelcol_test

import (
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
)

// exporter returns the config's exporter of that name, empty where it has none.
func exporter(config map[string]any, name string) map[string]any {
	all, _ := config["exporters"].(map[string]any)
	got, _ := all[name].(map[string]any)
	return got
}

// SigNoz takes the collector's data by gRPC on loopback, with no TLS, from every pipeline but the
// one that sends claude's metrics elsewhere.
func TestCollectorSendsToSigNozByGRPC(t *testing.T) {
	m, e := machine(t, variable+"=https://c.example:4317\n")
	if _, err := step.Run(t.Context(), kind.Steps(collector(t, e)), e.Out, false); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(m.Files[config], &parsed); err != nil {
		t.Fatal(err)
	}
	signoz := exporter(parsed, "otlp_grpc/signoz")
	tls, _ := signoz["tls"].(map[string]any)
	if signoz["endpoint"] != "127.0.0.1:14317" || tls["insecure"] != true {
		t.Errorf("exporter otlp_grpc/signoz = %v, want 127.0.0.1:14317 with no TLS", signoz)
	}
	if strings.Contains(string(m.Files[config]), "otlp_http") {
		t.Error("the config still sends by OTLP/HTTP")
	}
	pipelines, _ := parsed["service"].(map[string]any)["pipelines"].(map[string]any)
	for name, p := range pipelines {
		want := []any{"otlp_grpc/signoz"}
		if name == "metrics/claude" {
			want = []any{"otlp_grpc/claude"}
		}
		if got := p.(map[string]any)["exporters"]; !slices.Equal(got.([]any), want) {
			t.Errorf("pipeline %s exports to %v, want %v", name, got, want)
		}
	}
}
