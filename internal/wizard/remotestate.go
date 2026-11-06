package wizard

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/zadykian/ws/internal/server/apply"
)

// ErrOtherWs says that ws holds another version of ws, or none, as before the stage that installs
// this one. Another version's report names its own steps, and an old one has no report to give.
var ErrOtherWs = errors.New("ws's report refused")

// State returns the roles of what ws server state --json reports on ws, from a ws of this version
// alone. It runs no state op where Status names another, and the report's own version decides after
// that. The op changes nothing, so what Status read stands.
func (r *Remote) State(ctx context.Context, version string) ([]apply.RoleState, error) {
	if !r.Status(ctx) {
		return nil, errors.New("ws's report: ws's status read nothing")
	}
	if held := r.Get("ws"); held != version {
		return nil, otherWs(held, version)
	}
	out, err := r.query(ctx, "state")
	if err != nil {
		return nil, fmt.Errorf("ws's report: %w", err)
	}
	var doc apply.StateDoc
	if err := json.Unmarshal(out, &doc); err != nil {
		return nil, fmt.Errorf("ws's report: %w", err)
	}
	if doc.Version != version {
		return nil, otherWs(doc.Version, version)
	}
	return doc.Roles, nil
}

// otherWs is ErrOtherWs, with the version that ws holds, - for none, and this ws's.
func otherWs(held, version string) error {
	return fmt.Errorf("%w: ws holds %s, and this ws is %s", ErrOtherWs, held, version)
}
