package claude

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

// telemetryRemoved are the env keys that Telemetry removes. A signal's own endpoint or protocol
// would override the general one. The temporality key changes how claude sends its counters. The
// Claude dashboard counts deltas, claude's default, and would show less of cumulative sums
// (internal/server/signoz/dashboards/README.md).
var telemetryRemoved = []string{
	"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
	"OTEL_EXPORTER_OTLP_TRACES_PROTOCOL",
	"OTEL_EXPORTER_OTLP_METRICS_ENDPOINT",
	"OTEL_EXPORTER_OTLP_METRICS_PROTOCOL",
	"OTEL_EXPORTER_OTLP_LOGS_ENDPOINT",
	"OTEL_EXPORTER_OTLP_LOGS_PROTOCOL",
	"OTEL_EXPORTER_OTLP_METRICS_TEMPORALITY_PREFERENCE",
}

// Telemetry sends claude's telemetry to the host's collector. The keys are in claude's user
// settings, not in .bashrc, so that the sessions cld restore sessions starts get them too. The
// file is written only where env changes: claude reorders the keys as it writes the file, so its
// bytes would differ at every run.
type Telemetry struct {
	M    effect.Machine
	User host.User
	// Env is the keys to set, with their values.
	Env map[string]string
}

func (t Telemetry) path() string { return t.User.Home + "/.claude/settings.json" }

type writeSettings struct {
	t       Telemetry
	data    []byte
	edit    envEdit
	created bool
}

// String names the keys set and removed, never the file's other content: it may hold tokens.
func (c writeSettings) String() string {
	text := "set claude's telemetry in " + c.t.path()
	if c.created {
		text = "create " + c.t.path() + " with claude's telemetry"
	}
	if len(c.edit.Set) > 0 {
		text += "\n  set: " + strings.Join(c.edit.Set, ", ")
	}
	if len(c.edit.Removed) > 0 {
		text += "\n  remove: " + strings.Join(c.edit.Removed, ", ")
	}
	return text
}

// Apply writes as the user, through a link as into a clone of dotfiles. A link, or a file that the
// user plants, then reaches only what the user could write. A new file gets claude's mode, and an
// existing one keeps its own. A missing ~/.claude is made at 0755, the user's.
func (c writeSettings) Apply(ctx context.Context) error {
	return c.t.User.PassInput(ctx, c.t.M, c.data, "sh", "-c", `umask 022 && mkdir -p "${1%/*}" && cat >"$1"`, "sh", c.t.path())
}

// Plan edits the settings in memory, and plans a write where env differs.
func (t Telemetry) Plan(ctx context.Context) ([]host.Change, error) {
	data, err := readAsUser(ctx, t.M, t.User, t.path())
	created := errors.Is(err, fs.ErrNotExist)
	if err != nil && !created {
		return nil, err
	}
	out, edit, err := mergeEnv(data, t.Env, telemetryRemoved)
	if err != nil {
		return nil, fmt.Errorf("%s %w", t.path(), err)
	}
	if edit.none() && !created {
		return nil, nil
	}
	return []host.Change{writeSettings{t, out, edit, created}}, nil
}
