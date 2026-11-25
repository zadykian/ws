package otelcol_test

import (
	"slices"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/values"
)

// exporter returns the config's exporter of that name, empty where it has none.
func exporter(config map[string]any, name string) map[string]any {
	all, _ := config["exporters"].(map[string]any)
	got, _ := all[name].(map[string]any)
	return got
}

// rendered runs the collector's service over the side's values and returns the config it writes.
func rendered(t *testing.T, data any, env string) map[string]any {
	m, e := machineOf(t, env, data)
	if _, err := step.Run(t.Context(), kind.Steps(collector(t, e)), e.Out, false); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(m.Files[config], &parsed); err != nil {
		t.Fatalf("the config is no YAML: %v", err)
	}
	return parsed
}

// SigNoz takes the collector's data by gRPC with no TLS, at the one endpoint that each side's values
// give. That is the loopback intake on the host and in a self-contained guest, and the socket in
// any other.
func TestCollectorSendsToSigNozAtItsSidesEndpoint(t *testing.T) {
	guest, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	self, err := values.Parse([]byte("WS_SELF_CONTAINED: true\n"))
	if err != nil {
		t.Fatal(err)
	}
	host, err := values.ParseHost(nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		data any
		want string
	}{
		"a guest under a host":   {guest, "unix:///var/lib/ws/otlp/signoz.sock"},
		"a self-contained guest": {self, "127.0.0.1:14317"},
		"the host":               {host, "127.0.0.1:14317"},
	} {
		signoz := exporter(rendered(t, c.data, ""), "otlp_grpc/signoz")
		tls, _ := signoz["tls"].(map[string]any)
		if signoz["endpoint"] != c.want || tls["insecure"] != true {
			t.Errorf("%s: exporter otlp_grpc/signoz = %v, want %s with no TLS", name, signoz, c.want)
		}
	}
}

// Temperatures and fans are the host's: a guest's sysfs shows the host's hwmon, so a guest would send
// them under its own name.
func TestOnlyTheHostScrapesNodeExporter(t *testing.T) {
	guest, _ := values.Parse(nil)
	host, _ := values.ParseHost(nil)
	for name, c := range map[string]struct {
		data any
		want bool
	}{"a guest": {guest, false}, "the host": {host, true}} {
		parsed := rendered(t, c.data, "")
		receivers, _ := parsed["receivers"].(map[string]any)
		_, has := receivers["prometheus"]
		metrics := parsed["service"].(map[string]any)["pipelines"].(map[string]any)["metrics"].(map[string]any)["receivers"].([]any)
		if has != c.want || slices.Contains(metrics, any("prometheus")) != c.want {
			t.Errorf("%s: the prometheus receiver is defined %v, in the pipeline %v; want %v", name, has, metrics, c.want)
		}
	}
}

// SigNoz takes the collector's data from every pipeline but the one that sends claude's metrics
// elsewhere.
func TestCollectorSendsEachPipelineToSigNozButClaudes(t *testing.T) {
	m, e := machine(t, variable+"=https://c.example:4317\n")
	if _, err := step.Run(t.Context(), kind.Steps(collector(t, e)), e.Out, false); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(m.Files[config], &parsed); err != nil {
		t.Fatal(err)
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
