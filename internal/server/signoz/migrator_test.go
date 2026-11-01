package signoz

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
)

const lsLine = "docker container ls --all --filter name=^signoz-telemetrystore-migrator$ --format {{.Status}}"

func TestAwaitMigrator(t *testing.T) {
	for _, tc := range []struct{ name, status, fails string }{
		{"finished", "Exited (0) 2 minutes ago", ""},
		{"gone", "", ""},
		{"failed", "Exited (1) 2 minutes ago", "ended as"},
		{"stopped", "Dead", "ended as"},
		{"still running", "Up 5 seconds", "gave up after 3 tries"},
		{"created", "Created", "gave up after 3 tries"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := effecttest.NewMachine()
			m.Results[lsLine] = effecttest.Result{Out: []byte(tc.status + "\n")}
			w := Wait{Tries: 3, Delay: time.Second, Sleep: func(context.Context, time.Duration) error { return nil }}
			err := awaitMigrator(t.Context(), m, w)
			if tc.fails == "" && err != nil || tc.fails != "" && (err == nil || !strings.Contains(err.Error(), tc.fails)) {
				t.Errorf("status %q: err = %v, want one with %q", tc.status, err, tc.fails)
			}
		})
	}
}
