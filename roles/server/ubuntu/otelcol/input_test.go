package otelcol_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
)

// A plan through the secrets task writes no /etc/ws/env, which mise writes in the run alone. The
// units take the endpoint from the run's env, so the plan shows the change that the run makes. A
// run without the task, whose env lacks the input, reads the file and keeps it.
func TestAPlanTakesTheRunsEndpoint(t *testing.T) {
	m, e := machine(t, "export "+variable+"='https://c.example:4317'\n")
	if _, err := step.Run(t.Context(), kind.Steps(collector(t, e)), e.Out, false); err != nil {
		t.Fatal(err)
	}
	e.Hidden = []string{variable}
	for _, c := range []struct {
		env  map[string]string
		want []string
	}{
		{map[string]string{}, nil},
		{map[string]string{variable: "https://c.example:4317"}, nil},
		{map[string]string{variable: "https://d.example:4317"}, []string{"write " + envFile + " (content not shown)"}},
		{map[string]string{variable: ""}, []string{"remove " + envFile, "write " + config}},
	} {
		e.LookupEnv = func(k string) (string, bool) {
			v, ok := c.env[k]
			return v, ok
		}
		p, err := collector(t, e)[0].Step.Plan(t.Context())
		if err != nil {
			t.Fatalf("env %q: %v", c.env, err)
		}
		var got []string
		for _, ch := range p.Changes {
			line := step.Headline(ch)
			if strings.Contains(step.PendingOf(ch).Text, "d.example") {
				t.Errorf("env %q: the plan shows the endpoint in %q", c.env, line)
			}
			if !strings.Contains(line, "restart") {
				got = append(got, line)
			}
		}
		slices.Sort(got)
		if !slices.Equal(got, c.want) {
			t.Errorf("env %q: plan = %q, want %q", c.env, got, c.want)
		}
	}
	if !strings.Contains(string(m.Files[envFile]), "c.example") {
		t.Error("a plan wrote claude.env")
	}
}
