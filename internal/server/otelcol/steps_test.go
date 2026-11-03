package otelcol

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

const validate = "env OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT=http://remote.invalid:4317 /usr/bin/otelcol-contrib validate --config=/etc/otelcol-contrib/config.yaml.ws-check"

func fakeSteps(t *testing.T, env string) (*effecttest.Machine, []step.Step) {
	m := machine(version + " ii\n")
	m.Dirs["/run/systemd/system"] = 0o755
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n prometheus-node-exporter"] = effecttest.Result{Out: []byte("prometheus-node-exporter ii\n")}
	m.Results[validate] = effecttest.Result{}
	if env != "" {
		m.Files["/home/ws/.secrets/env"] = []byte(env)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	t.Cleanup(srv.Close)
	v := values.Default()
	v.User, v.Home = "ws", "/home/ws"
	return m, steps(v, m, &effecttest.Units{}, &effecttest.Out{}, srv.URL)
}

func TestStepsWaitOnTheirPredecessorsInOrder(t *testing.T) {
	_, ss := fakeSteps(t, "")
	var names, after []string
	for _, s := range ss {
		names, after = append(names, s.Name()), append(after, s.WaitsOn())
	}
	wantNames := []string{exporterStep, downloadStep, installStep, configStep, healthStep}
	wantAfter := []string{"", "", downloadStep, installStep, configStep}
	if !slices.Equal(names, wantNames) || !slices.Equal(after, wantAfter) {
		t.Errorf("steps = %q after %q, want %q after %q", names, after, wantNames, wantAfter)
	}
}

func TestStepsApplyOnceAndThenPlanNothing(t *testing.T) {
	m, ss := fakeSteps(t, "OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT='https://remote.example:4317'\n")
	out := &effecttest.Out{}
	if changed, err := step.Run(t.Context(), ss, out, false); err != nil || !changed {
		t.Fatalf("first run = %v, %v; want changes", changed, err)
	}
	if changed, err := step.Run(t.Context(), ss, out, false); err != nil || changed {
		t.Errorf("second run = %v, %v; want nothing to change", changed, err)
	}
	for path, want := range map[string]string{
		"/etc/default/prometheus-node-exporter":                     "127.0.0.1:9100",
		"/etc/otelcol-contrib/claude.env":                           "https://remote.example:4317",
		"/etc/otelcol-contrib/config.yaml":                          "otlp",
		"/etc/systemd/system/otelcol-contrib.service.d/claude.conf": "claude.env",
	} {
		if !strings.Contains(string(m.Files[path]), want) {
			t.Errorf("%s = %q, want it to hold %q", path, m.Files[path], want)
		}
	}
}

func TestStepsRemoveTheEndpointWhereNoneIsSet(t *testing.T) {
	m, ss := fakeSteps(t, "")
	m.Files["/etc/otelcol-contrib/claude.env"] = []byte("old")
	if _, err := step.Run(t.Context(), ss, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.Files["/etc/otelcol-contrib/claude.env"]; ok {
		t.Error("claude.env stays where no endpoint is set")
	}
}

func TestStepsRefuseALoopingEndpoint(t *testing.T) {
	_, ss := fakeSteps(t, "OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT=http://127.0.0.1:4317\n")
	_, err := step.Run(t.Context(), ss, &effecttest.Out{}, true)
	if err == nil || strings.Contains(err.Error(), "127.0.0.1") {
		t.Errorf("error = %v, want a refusal that leaves the value out", err)
	}
}
