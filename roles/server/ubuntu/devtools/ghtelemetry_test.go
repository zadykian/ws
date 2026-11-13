package devtools

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
)

const (
	asWS         = "runuser -u ws -- env -C /home/ws HOME=/home/ws env GH_TELEMETRY=0 gh config "
	telemetryGet = asWS + "get telemetry"
	telemetrySet = asWS + "set telemetry disabled"
)

func TestGHTelemetryPlan(t *testing.T) {
	tests := []struct {
		name string
		held effecttest.Result
		want int
	}{
		{"disabled", effecttest.Result{Out: []byte("disabled\n")}, 0},
		{"enabled, gh's default", effecttest.Result{Out: []byte("enabled\n")}, 1},
		{"no gh yet", effecttest.Result{Err: errors.New("exit 127")}, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := effecttest.NewMachine()
			m.Results[telemetryGet], m.Results[telemetrySet] = tt.held, effecttest.Result{}
			g := GHTelemetry{M: m, User: account.User{Name: "ws", Home: "/home/ws"}}
			changes, err := g.Plan(context.Background())
			if err != nil || len(changes) != tt.want {
				t.Fatalf("got %v, %v; want %d changes", changes, err, tt.want)
			}
			apply(t, changes)
			if set := slices.Contains(m.Calls, "root "+telemetrySet); set != (tt.want == 1) {
				t.Errorf("calls %q", m.Calls)
			}
		})
	}
}

func apply(t *testing.T, changes []resource.Change) {
	for _, c := range changes {
		if err := c.Apply(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
}
