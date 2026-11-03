package signoz

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/zadykian/ws/effect"
)

// migrator is the schema migrator's container, a one-shot service that each up of compose runs.
const migrator = "signoz-telemetrystore-migrator"

var (
	migratorBusy = regexp.MustCompile(`^(Created|Up|Restarting)`)
	migratorDone = regexp.MustCompile(`^Exited \(0\)`)
)

// awaitMigrator waits for the schema migrator, which makes the tables after SigNoz answers and
// sets some of their retention to its own default as it goes. A change beside it may also fail on
// a column it has yet to add, while SigNoz reports it as done. A migrator that is gone, as a prune
// removes stopped containers, finished on an earlier run. It fails where the last run failed, or
// was stopped before it finished.
func awaitMigrator(ctx context.Context, m effect.Machine, w Wait) error {
	w = w.orDefault(120, 5*time.Second)
	return w.poll(ctx, migrator, func() (bool, string, error) {
		out, err := m.RunRoot(ctx, "docker", "container", "ls", "--all",
			"--filter", "name=^"+migrator+"$", "--format", "{{.Status}}")
		if err != nil {
			return false, "", err
		}
		status := strings.TrimSpace(string(out))
		switch {
		case migratorBusy.MatchString(status):
			return false, status, nil
		case status != "" && !migratorDone.MatchString(status):
			return false, "", fmt.Errorf("%s: its last run ended as %q", migrator, status)
		}
		return true, "", nil
	})
}
